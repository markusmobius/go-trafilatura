import argparse
import ast
import configparser
import importlib.metadata
import json
import re
import subprocess
from pathlib import Path
from types import SimpleNamespace

from lxml import etree
from trafilatura.settings import Extractor


COMMIT = "c1bc9531a2a978326112ca9987e1382745116136"
ROOT = Path(__file__).resolve().parents[2]
OUTPUTS = {}


class Expression:
    def __init__(self, value):
        self.value = value

    def __getattr__(self, name):
        if name in ("startswith", "endswith", "count", "strip", "replace", "get"):
            return lambda *arguments: Expression({"op": name, "args": [self.value, *[encode(argument) for argument in arguments]]})
        return Expression({"op": "field", "args": [self.value, name]})

    def __getitem__(self, key):
        return Expression({"op": "field", "args": [self.value, encode(key)]})

    def __iter__(self):
        raise TypeError("An upstream result must not be iterated while importing assertions")


def encode(value):
    if isinstance(value, Expression):
        return value.value
    if isinstance(value, Metadata):
        return value.expression.value
    if isinstance(value, etree._Element):
        return etree.tostring(value, encoding="unicode")
    if isinstance(value, configparser.ConfigParser):
        return dict(value["DEFAULT"])
    if isinstance(value, Extractor):
        return {name: encode(getattr(value, name)) for name in ("focus", "links", "images", "tables", "comments")}
    if isinstance(value, re.Pattern):
        return value.pattern
    if isinstance(value, set):
        return [encode(item) for item in sorted(value)]
    if isinstance(value, (tuple, list)):
        return [encode(item) for item in value]
    if isinstance(value, dict):
        return {key: encode(item) for key, item in value.items()}
    return value


class Metadata:
    def __init__(self):
        self.initial = {}
        self.expression = Expression({"literal": {}})

    def __getattr__(self, name):
        if name == "__slots__":
            return ("title", "author", "url", "hostname", "description", "sitename", "date", "categories", "tags", "license", "image", "pagetype")
        return getattr(self.expression, name)

    def as_dict(self):
        return self.expression


class Recorder:
    def __init__(self):
        self.test = ""
        self.operations = []
        self.assertions = []
        self.unmapped = []

    def call(self, operation, *arguments, **keywords):
        initial = {}
        for argument in arguments:
            if isinstance(argument, Metadata):
                initial = {"base": argument.expression.value, "fields": {key: encode(value) for key, value in vars(argument).items() if key not in ("initial", "expression")}}
        entry = {"test": self.test, "operation": operation, "arguments": [encode(value) for value in arguments if not isinstance(value, Metadata)], "keywords": encode(keywords), "initial": initial}
        index = len(self.operations)
        self.operations.append(entry)
        expression = Expression({"result": index})
        if operation in ("extract_metadata", "extract_meta_json", "extract_json", "extract_json_parse_error", "process_parent"):
            result = next((value for value in arguments if isinstance(value, Metadata)), Metadata())
            result.expression = expression
            for name in list(vars(result)):
                if name not in ("initial", "expression"):
                    delattr(result, name)
            return result
        return expression

    def check(self, value, line, source):
        expression = encode(value)
        def contains_result(item):
            if isinstance(item, dict):
                return "result" in item or any(contains_result(child) for child in item.values())
            if isinstance(item, list):
                return any(contains_result(child) for child in item)
            return False
        if not contains_result(expression):
            self.unmapped.append({"test": self.test, "line": line, "assertion": source})
            return
        self.assertions.append({"test": self.test, "line": line, "source": source, "expression": expression})


class Assertions(ast.NodeTransformer):
    def __init__(self, native=None):
        self.native = native or {}

    def visit_Assert(self, node):
        if node.lineno in self.native:
            return ast.copy_location(ast.Pass(), node)
        source = ast.unparse(node.test)
        value = self.expression(node.test)
        return ast.copy_location(ast.Expr(value=ast.Call(func=ast.Name(id="_check", ctx=ast.Load()), args=[value, ast.Constant(node.lineno), ast.Constant(source)], keywords=[])), node)

    def expression(self, node):
        if isinstance(node, ast.Compare):
            operands = [node.left, *node.comparators]
            pieces = [self.operation(type(operator).__name__, operands[index:index + 2]) for index, operator in enumerate(node.ops)]
            return pieces[0] if len(pieces) == 1 else self.operation("And", pieces)
        if isinstance(node, ast.BoolOp):
            return self.operation(type(node.op).__name__, node.values)
        if isinstance(node, ast.UnaryOp) and isinstance(node.op, ast.Not):
            return self.operation("Not", [node.operand])
        if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and node.func.id in ("len", "str", "all", "any"):
            if node.func.id in ("len", "str"):
                return self.operation(node.func.id, node.args)
            return ast.Call(func=ast.Name(id="_" + node.func.id, ctx=ast.Load()), args=[self.expression(argument) for argument in node.args], keywords=[])
        for field, value in ast.iter_fields(node):
            if isinstance(value, ast.AST):
                setattr(node, field, self.expression(value))
            elif isinstance(value, list):
                setattr(node, field, [self.expression(item) if isinstance(item, ast.AST) else item for item in value])
        return node

    def operation(self, operator, arguments):
        return ast.Call(func=ast.Name(id="_operation", ctx=ast.Load()), args=[ast.Constant(operator), *[self.expression(argument) for argument in arguments]], keywords=[])


def operation(name, *arguments):
    return Expression({"op": name, "args": [encode(argument) for argument in arguments]})


def aggregate(name, values):
    ordered = sorted(values, key=lambda value: json.dumps(encode(value), sort_keys=True))
    return operation(name, *ordered)


def import_realworld(upstream):
    filename = "realworld_tests.py"
    source = subprocess.check_output(["git", "-C", str(upstream), "show", f"{COMMIT}:tests/{filename}"], encoding="utf-8")
    tree = ast.parse(source)
    functions = [node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name.startswith("test_")]
    counts = {node.name: sum(isinstance(child, ast.Assert) for child in ast.walk(node)) for node in functions}
    formatted_only = [{"line": node.lineno, "source": ast.unparse(node.test), "reason": "Excluded: formatted-text serializer variant; the same saved page has an imported unformatted assertion."} for node in ast.walk(tree) if isinstance(node, ast.Assert) and node.lineno in (678, 840, 845)]
    recorder = Recorder()
    namespace = {"__file__": str(upstream / "tests" / filename), "__name__": "upstream_test_import"}
    exec(compile(ast.fix_missing_locations(Assertions().visit(tree)), filename, "exec"), namespace)
    namespace.update({"_check": recorder.check, "_operation": operation, "_all": lambda values: aggregate("And", values), "_any": lambda values: aggregate("Or", values)})
    namespace["load_mock_page"] = lambda *args, **kwargs: recorder.call("load_mock_page", *args, **kwargs)
    namespace["load_mock_page_meta"] = lambda url: recorder.call("load_fixture", namespace["MOCK_PAGES"][url])
    namespace["extract_metadata"] = lambda *args, **kwargs: recorder.call("extract_metadata", *args, **kwargs)
    inventory = []
    for function in functions:
        recorder.test = filename + "/" + function.name
        exclusion = ""
        if function.name == "test_extract_links_formatting":
            exclusion = "The two Markdown syntax assertions are translated to link/emphasis DOM assertions in Test_Python220_RealWorldFormatting."
        targets = ["Test_Python220_RealWorldFormatting"] if function.name == "test_extract_links_formatting" else []
        inventory.append({"test": recorder.test, "line": function.lineno, "assertions": counts[function.name], "exclusion": exclusion, "go_tests": targets, "native": formatted_only if function.name == "test_extract" else []})
        if not exclusion:
            if function.name == "test_extract":
                namespace[function.name](xmloutput=False, formatting=False)
                for variant in ("xml", "formatted_text"):
                    inventory.append({"test": recorder.test + "/" + variant, "line": function.lineno, "assertions": counts[function.name], "exclusion": "Python XML/Markdown serialization is excluded; the unformatted text parameterization tests the same saved pages."})
            else:
                namespace[function.name]()
    write_suite("realworld", recorder, inventory)


def write_suite(name, recorder, inventory):
    output = {"commit": COMMIT, "inventory": inventory, "operations": recorder.operations, "assertions": recorder.assertions, "unmapped": recorder.unmapped}
    destination = ROOT / "test-files" / f"python-2.2.0-{name}.json"
    OUTPUTS[destination] = (json.dumps(output, ensure_ascii=True, indent=2) + "\n").encode("utf-8")
    print(f"Imported {len(recorder.operations)} operations and {len(recorder.assertions)} assertions into {destination}")
    print("Assertions requiring native adapters:", json.dumps(recorder.unmapped, ensure_ascii=True))


def import_units(upstream, structural=False):
    filename = "unit_tests.py"
    source = subprocess.check_output(["git", "-C", str(upstream), "show", f"{COMMIT}:tests/{filename}"], encoding="utf-8")
    tree = ast.parse(source)
    selected = {
        "test_precision_recall",
        "test_no_duplicate_content_short_elements",
        "test_recall_escalation_justext",
        "test_escalation_retry_no_comment_capture",
        "test_loose_text_tail_not_squished",
        "test_mixed_content_extraction",
        "test_nonstd_html_entities",
        "test_markdown_empty_sup_sub_are_dropped",
    }
    if structural:
        selected = {"test_code_blocks", "test_yoast_faq_block", "test_exotic_tags", "test_formatting", "test_external", "test_images", "test_links", "test_htmlprocessing"}
    native = {}
    for lines, reason, targets in (
        ((317, 1235), "Excluded: Python TEI serialization and validation.", []),
        ((325, 326, 334), "Native helper assertions: Test_ExoticTags.", ["Test_ExoticTags"]),
        ((417, 418, 419, 432, 439, 451, 529, 538, 541, 567, 607, 610), "Markdown serialization excluded; inline/code/list DOM behavior is checked by Test_Python220_Structures, Test_Python220_CodeAndFAQ, and Test_Formatting.", []),
        ((425,), "Excluded: Python Markdown/YAML metadata serialization.", []),
        ((506, 510), "Native formatting/container assertions: Test_Formatting.", ["Test_Formatting"]),
        ((812, 815, 821, 828), "Native fallback sanitization assertions: Test_External.", ["Test_External"]),
        ((886, 887, 895, 896, 898, 899, 900, 906, 910, 914, 918), "Image/URL DOM translation: Test_Python220_ImagesAndLinks.", ["Test_Python220_ImagesAndLinks"]),
        ((951, 952), "Native empty-link and formatting assertions: Test_Links.", ["Test_Links"]),
        ((961, 965, 972, 983, 989), "Link/URL/license DOM translation: Test_Python220_ImagesAndLinks.", ["Test_Python220_ImagesAndLinks"]),
        ((1203, 1316, 1321), "Excluded: Python-only xmltotxt/replace_element_text serialization.", []),
        ((1206, 1207, 1213, 1228, 1232, 1270, 1271, 1274, 1275, 1278, 1279, 1282, 1283, 1286, 1289, 1292, 1295, 1299, 1302, 1307, 1309, 1326, 1332, 1336), "Native DOM/helper assertions: Test_Python220_HTML and Test_Python220_Internals.", ["Test_Python220_HTML", "Test_Python220_Internals"]),
        ((1242, 1245, 1250, 1254), "Native etree manipulation assertions: Test_HtmlProcessing.", ["Test_HtmlProcessing"]),
    ):
        for line in lines:
            native[line] = {"reason": reason, "go_tests": targets}
    functions = [node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name in selected]
    if {node.name for node in functions} != selected:
        raise ValueError("An inventoried extraction test is missing from the pinned source")
    counts = {node.name: sum(isinstance(child, ast.Assert) for child in ast.walk(node)) for node in functions}
    alternatives = {node.name: [{"line": child.lineno, "source": ast.unparse(child.test), **native[child.lineno]} for child in ast.walk(node) if isinstance(child, ast.Assert) and child.lineno in native] for node in functions}
    recorder = Recorder()
    namespace = {"__file__": str(upstream / "tests" / filename), "__name__": "upstream_test_import"}
    exec(compile(ast.fix_missing_locations(Assertions(native).visit(tree)), filename, "exec"), namespace)
    namespace.update({"_check": recorder.check, "_operation": operation, "_all": lambda values: aggregate("And", values), "_any": lambda values: aggregate("Or", values)})
    namespace["extract"] = lambda *args, **kwargs: recorder.call("extract_dom" if structural else "extract", *args, **kwargs)
    namespace["load_mock_page"] = lambda *args, **kwargs: recorder.call("load_mock_page", *args, **kwargs)
    namespace["LANGID_FLAG"] = True
    if structural:
        for name in ("is_image_file", "handle_image", "handle_textelem"):
            namespace[name] = lambda *args, _name=name, **kwargs: recorder.call(_name, *args, **kwargs)
    inventory = []
    for function in functions:
        recorder.test = filename + "/" + function.name
        inventory.append({"test": recorder.test, "line": function.lineno, "assertions": counts[function.name], "exclusion": "", "native": alternatives[function.name]})
        if function.args.args:
            namespace[function.name](Extractor())
        else:
            namespace[function.name]()
    write_suite("structures" if structural else "extraction", recorder, inventory)


def write_coverage(upstream):
    imported = {}
    for destination, content in OUTPUTS.items():
        suite = json.loads(content)
        for entry in suite["inventory"]:
            if entry["test"].count("/") != 1:
                continue
            imported[entry["test"]] = (destination.name, entry, suite)
    native_groups = {
        "Test_Python220_Internals": """
            test_trim test_recover_wild_text_default_tags test_prune_boilerplate_table_after_nested
            test_prune_keep_teasers test_aria_layout_table_reclassified test_table_colgroup_no_crash
            test_sanitize_tree_th_dedup test_sanitize_tree_absolutizes_links test_settings_element_lists
        """,
        "Test_Python220_HTML": """
            test_link_density_tables_threshold test_link_density_tables_textless_links_kept
            test_link_density_short_link_list_kept test_link_density_large_link_farm_pruned
            test_link_density_whole_card_links_kept test_overall_discard_legacy_tokens
            test_overall_discard_matches_both_attributes test_precision_discard_link_token_only
            test_body_xpath_fulltext_class test_basic_cleaning_cookie_banner_scope
        """,
        "Test_Python220_InputsAndOptions": """
            test_input test_extract_with_metadata test_extraction_options test_wrong_language_discarded
            test_large_doc_performance test_lang_detection test_html_conversion
        """,
        "Test_Python220_Structures": """
            test_blockquote_inline_content test_list_item_block_child_single_bullet test_list_item_image_gets_bullet
            test_ordered_list_numbering test_nested_list_indentation test_list_item_attr_whitelist
            test_list_item_link_with_inline_formatting test_paragraph_link_with_inline_formatting
            test_nested_inline_formatting test_blockquote_bare_inline test_del_and_code_in_non_paragraph_contexts
            test_hi_del_nesting_with_direct_text test_image_tail_not_duplicated test_table_image_in_cell
            test_combined_links_formatting_images_tables test_combined_flags_toggle_off
            test_table_cell_keeps_nested_formatting test_include_images_does_not_truncate
        """,
        "Test_Python220_Tables": """
            test_table_colspan_content test_table_colspan_padding test_table_bad_span_attr_treated_as_colspan1
            test_table_huge_or_bad_colspan_no_crash test_colspan_zero_trust test_table_rowspan_aligned
            test_table_rowspan_colspan_combined test_table_rowspan_decrement_on_padding test_table_empty_cells_and_rows
            test_table_cell_block_elements_flattened test_table_nested_in_cell test_table_nested_in_cell_pipeline
            test_table_nested_tail_preserved test_table_nested_tail_with_prior_child test_table_comment_in_row
            test_table_caption test_table_orphan_cells_no_tr test_table_stray_cell_descendant
        """,
        "Test_Python220_Recovery": """
            test_no_duplicate_content test_no_duplicate_content_list_item test_no_duplicate_content_nonadjacent
            test_recover_wild_text_inline_formatting_dedup test_recall_escalation
            test_recall_escalation_justext_comment_scoping test_recall_escalation_no_comment_doubling
            test_dfp_long_opening_post_keeps_replies test_dfp_precision_keeps_posts
            test_recall_escalation_blog_comment_leak test_main_pass_excludes_comments_when_disabled
            test_main_pass_excludes_details_wrapped_comments
        """,
        "Test_Python220_TableAndListLegacy": "test_table_cell_list_no_row_break test_recover_wild_text_dedup_scan_cap",
        "Test_TableProcessing Test_Python220_TableAndListLegacy": "test_table_processing",
        "Test_ListProcessing Test_Python220_TableAndListLegacy": "test_list_processing",
    }
    native = {name: target.split() for target, names in native_groups.items() for name in names.split()}
    excluded_groups = {
        "Python output serializers and their private rendering helpers are excluded; extracted HTML structure is tested separately.": """
            test_xmltocsv test_tojson test_python_output test_tei test_markdown_metadata_yaml_safe
            test_include_formatting_markdown test_markdown_list_item_inline_spacing test_markdown_sup_sub_keep_boundary
            test_is_in_table_cell test_inline_marker_flanking_whitespace test_inline_marker_edge_cases
            test_heading_level_zero_trust test_markdown_escaping test_markdown_link_angle_bracket_targets
            test_xmltotxt_no_mutation test_math_conversion test_inline_edge_cases test_heading_inline_formatting
        """,
        "Python process-global cache/reset APIs do not exist in Go; per-extraction deduplication is tested separately.": "test_reset_caches",
        "Python Document constructor defaults have no matching Go constructor; Go allocates extraction state per call.": "test_document_isolation",
        "Go Options has no URL blacklist option.": "test_url_blacklist",
        "Python XPath regex compilation has no equivalent API; Go uses native selector predicates.": "test_xpath_alt_rejects_empty_group",
        "The bundled Python readability and jusText implementations are not ported; Go uses external fallback packages.": "test_compare_extraction_justext_ratio test_is_probably_readerable",
        "Python config-file, deprecation, and conflicting dynamic-option APIs are excluded; Go uses typed options.": "test_config_loading test_deprecations test_incompatible_options",
    }
    excluded = {name: reason for reason, names in excluded_groups.items() for name in names.split()}
    notes = {
        "test_input": "Reader/parser API translation. Encoding-candidate lists, repair_faulty_html strings, dynamic input types, source coercion, and Python output-format validation have no Go API. Nil and gzip expectations are retained.",
        "test_trim": "trim/textfilter assertions are native. Nullable sanitize and process-global cache assertions are explicitly skipped.",
        "test_extract_with_metadata": "Metadata and content assertions use Extract with Python-equivalent date settings; Python raw_text/fingerprint serialization is excluded.",
        "test_extraction_options": "Native size, metadata, language, and date checks. Python-only serializer errors, try_justext internals, external readability internals, and with_metadata=False are excluded.",
        "test_html_conversion": "End-to-end extracted HTML nodes, title, and image assertions are translated. Python's internal XML converter, document wrapper, fingerprint, and pretty-print serialization are excluded.",
        "test_table_processing": "Existing helper translations plus original missing cases in Test_Python220_TableAndListLegacy. Markdown separators/whitespace are represented by cell/header DOM assertions, not serializer parity.",
        "test_list_processing": "Existing helper translations plus original basic-order and link-only-item cases. List markers/indentation and description-item rend attributes use native HTML semantics.",
        "test_recover_wild_text_dedup_scan_cap": "Default-cap input is tested. Monkeypatching Go's constant to Python's two test caps is unavailable and explicitly skipped.",
        "test_combined_flags_toggle_off": "Image/link toggles are native; Go always preserves HTML formatting, so the formatting-off option is explicitly skipped.",
        "test_recover_wild_text_default_tags": "Go's function returns void; its no-crash behavior is checked, not Python's non-null return value.",
        "test_document_as_dict": "Metadata fields are projected from Go's typed struct, not a Python Document serialization API.",
        "test_dedup": "The four paragraph-level process_node assertions are translated. Cross-document process-global cache accumulation is not a Go API and is explicitly skipped.",
    }
    available = set()
    for file in ROOT.glob("*_test.go"):
        available.update(re.findall(r"func (Test_[A-Za-z0-9_]+)\(", file.read_text(encoding="utf-8")))
    modules = ("baseline_tests.py", "unit_tests.py", "metadata_tests.py", "json_metadata_tests.py", "filters_tests.py", "deduplication_tests.py", "realworld_tests.py")
    coverage = []
    missing = []
    for filename in modules:
        source = subprocess.check_output(["git", "-C", str(upstream), "show", f"{COMMIT}:tests/{filename}"], encoding="utf-8")
        for function in ast.parse(source).body:
            if not isinstance(function, ast.FunctionDef) or not function.name.startswith("test_"):
                continue
            identity = filename + "/" + function.name
            lines = sorted(node.lineno for node in ast.walk(function) if isinstance(node, ast.Assert))
            entry = {"test": identity, "line": function.lineno, "assertion_lines": lines}
            if identity in imported:
                fixture, original, suite = imported[identity]
                entry["fixture"] = fixture
                entry["recorded_cases"] = sum(check["test"] == identity for check in suite["assertions"])
                entry["imported_lines"] = sorted({check["line"] for check in suite["assertions"] if check["test"] == identity})
                entry["native_or_excluded"] = original.get("native", [])
                entry["unmapped"] = [check for check in suite["unmapped"] if check["test"] == identity]
                accounted = set(entry["imported_lines"]) | {check["line"] for check in entry["native_or_excluded"]} | {check["line"] for check in entry["unmapped"]}
                if original["exclusion"]:
                    entry["status"] = "excluded"
                    entry["note"] = original["exclusion"]
                    if function.name == "test_extract_links_formatting":
                        entry["status"] = "translated"
                        entry["go_tests"] = ["Test_Python220_RealWorldFormatting"]
                else:
                    entry["status"] = "mixed" if entry["native_or_excluded"] or entry["unmapped"] else "imported"
                    entry["unaccounted_lines"] = sorted(set(lines) - accounted)
                    if entry["unaccounted_lines"]:
                        missing.append(identity + ": " + str(entry["unaccounted_lines"]))
            elif filename == "unit_tests.py" and function.name in excluded:
                entry.update(status="excluded", note=excluded[function.name])
            else:
                targets = native.get(function.name) if filename == "unit_tests.py" else None
                if filename == "baseline_tests.py":
                    targets = ["Test_Python220_Baseline", "Test_Baseline"]
                    entry["note"] = "Native baseline translation, including the existing long article fixture. Go's parsed-DOM API replaces Python raw/dynamic inputs; Python-only cap monkeypatching is explicitly skipped."
                elif filename == "filters_tests.py":
                    targets = ["Test_Python220_Filters"]
                    entry["note"] = "Language-enabled branch; zero-size Python config is matched. XPath element selectors become CSS; comment XPath, URL blacklist, and config-file APIs are explicitly excluded."
                elif filename == "deduplication_tests.py":
                    targets = ["Test_Python220_Deduplication"]
                    if function.name not in ("test_lrucache", "test_dedup"):
                        entry.update(status="excluded", note="Standalone hashes, fingerprints, tokens, Simhash, and process-global cache/reset APIs are excluded.")
                        targets = None
                if targets:
                    entry.update(status="translated", go_tests=targets)
                elif "status" not in entry:
                    missing.append(identity)
            if function.name in notes:
                entry["note"] = notes[function.name]
            targets = set(entry.get("go_tests", []))
            for mapping in entry.get("native_or_excluded", []):
                targets.update(mapping.get("go_tests", []))
            for target in targets:
                if target not in available:
                    raise ValueError(f"Inventoried Go test is absent: {target}")
            coverage.append(entry)
    if missing:
        raise ValueError("Uninventoried source assertions: " + ", ".join(missing))
    output = {
        "commit": COMMIT,
        "status_definitions": {
            "imported": "Original recorded assertion expressions; Go API and DOM vocabulary adapters still apply.",
            "mixed": "Original recorded assertions plus source-line mappings for native translations or exclusions.",
            "translated": "Native behavioral translation; not a claim of identical Python API, serializer, input representation, or assertion count.",
            "excluded": "No supported Go contract in the agreed port scope; reason is recorded.",
        },
        "excluded_modules": {name: "Page acquisition/CLI or Python XML/TEI serialization is outside the agreed scope." for name in ("cli_tests.py", "downloads_tests.py", "feeds_tests.py", "sitemaps_tests.py", "spider_tests.py", "xml_tei_tests.py")},
        "tests": coverage,
    }
    destination = ROOT / "test-files" / "python-2.2.0-coverage.json"
    OUTPUTS[destination] = (json.dumps(output, ensure_ascii=True, indent=2) + "\n").encode("utf-8")
    print(f"Inventoried {len(coverage)} source test functions, including explicit translations and exclusions")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--upstream", type=Path, required=True)
    parser.add_argument("--check", action="store_true", help="Verify generated fixtures without writing files")
    arguments = parser.parse_args()
    if importlib.metadata.version("trafilatura") != "2.2.0":
        raise SystemExit("Fixture generation requires the pinned Trafilatura 2.2.0 reference environment")
    revision = subprocess.check_output(["git", "-C", str(arguments.upstream), "rev-parse", "v2.2.0"], text=True).strip()
    if revision != COMMIT:
        raise SystemExit("The upstream v2.2.0 tag does not match the pinned commit")
    revision = subprocess.check_output(["git", "-C", str(arguments.upstream), "rev-parse", "HEAD"], text=True).strip()
    if revision != COMMIT or subprocess.call(["git", "-C", str(arguments.upstream), "diff", "--quiet", COMMIT, "--", "tests/resources", "tests/eval"]) != 0:
        raise SystemExit("Use a checkout of the pinned commit with unchanged tracked test resources")
    recorder = Recorder()
    excluded = {
        "test_json_minify": "Python regex preprocessing has no separate Go API; escaped-name behavior is tested through JSON extraction.",
        "test_extract_json_processes_list_once": "Monkeypatch counts Python process_parent invocations; Go has a different traversal implementation.",
    }
    inventory = []
    operations = ("extract_metadata", "extract_meta_json", "extract_json", "extract_json_parse_error", "process_parent", "extract_title", "extract_url", "extract_metainfo", "normalize_tags", "check_authors", "normalize_authors", "extract_json_author", "normalize_json")
    for filename in ("metadata_tests.py", "json_metadata_tests.py"):
        source = subprocess.check_output(["git", "-C", str(arguments.upstream), "show", f"{COMMIT}:tests/{filename}"], encoding="utf-8")
        tree = ast.parse(source)
        functions = [node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name.startswith("test_")]
        counts = {node.name: sum(isinstance(child, ast.Assert) for child in ast.walk(node)) for node in functions}
        namespace = {"__file__": str(arguments.upstream / "tests" / filename), "__name__": "upstream_test_import"}
        exec(compile(ast.fix_missing_locations(Assertions().visit(tree)), filename, "exec"), namespace)
        namespace.update({"_check": recorder.check, "_operation": operation, "_all": lambda values: aggregate("And", values), "_any": lambda values: aggregate("Or", values), "Document": Metadata, "html": SimpleNamespace(fromstring=lambda value: value), "XPath": lambda value: value})
        for name in operations:
            namespace[name] = lambda *args, _name=name, **kwargs: recorder.call(_name, *args, **kwargs)
        for function in functions:
            recorder.test = filename + "/" + function.name
            inventory.append({"test": recorder.test, "line": function.lineno, "assertions": counts[function.name], "exclusion": excluded.get(function.name, "")})
            if function.name not in excluded:
                namespace[function.name]()
    write_suite("metadata", recorder, inventory)
    import_realworld(arguments.upstream)
    import_units(arguments.upstream)
    import_units(arguments.upstream, structural=True)
    write_coverage(arguments.upstream)
    compressed = subprocess.check_output(["git", "-C", str(arguments.upstream), "show", f"{COMMIT}:tests/resources/webpage.html.gz"])
    destination = ROOT / "test-files" / "mock" / "webpage.html.gz"
    if destination.exists() and destination.read_bytes() != compressed:
        raise ValueError(f"Refusing to replace a changed upstream fixture: {destination}")
    OUTPUTS[destination] = compressed
    for destination, content in OUTPUTS.items():
        unchanged = destination.exists() and destination.read_bytes() == content
        if arguments.check:
            if not unchanged:
                raise ValueError(f"Generated fixture is stale or missing: {destination}")
        elif not unchanged:
            destination.write_bytes(content)
    print("Generated fixtures verified" if arguments.check else "Generated fixtures updated")


if __name__ == "__main__":
    main()