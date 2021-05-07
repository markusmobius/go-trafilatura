# pylint:disable-msg=E0611,I1101
"""
Module bundling all functions needed to extract the text in a webpage.
"""

# This file is available from https://github.com/adbar/trafilatura
# under GNU GPL v3 license


# standard
import logging
import re  # import regex as re

from collections import OrderedDict
from copy import deepcopy

from lxml import etree, html

# own
from .external import justext_rescue, sanitize_tree, SANITIZED_XPATH, try_readability
from .filters import (check_html_lang, content_fingerprint, duplicate_test,
                      language_filter, text_chars_test)
from .htmlprocessing import (convert_tags, discard_unwanted,
                             discard_unwanted_comments, handle_textnode,
                             link_density_test, link_density_test_tables,
                             process_node, tree_cleaning)
from .metadata import extract_metadata, METADATA_LIST
from .settings import use_config, DEFAULT_CONFIG, TAG_CATALOG
from .utils import load_html, trim, txttocsv, is_image_file
from .xml import (build_json_output, build_xml_output, build_tei_output,
                  control_xml_output, xmltotxt)
from .xpaths import BODY_XPATH, COMMENTS_XPATH  # , REMOVE_COMMENTS_XPATH


LOGGER = logging.getLogger(__name__)


def baseline(filecontent):
    """Use baseline extraction function targeting text paragraphs and/or JSON metadata.

    Args:
        filecontent: HTML code as binary string or string.

    Returns:
        A LXML <body> element containing the extracted paragraphs,
        the main text as string, and its length as integer.

    """
    tree = load_html(filecontent)
    postbody = etree.Element('body')
    if tree is None:
        return postbody, 0, ''
    # scrape from json text
    for elem in tree.iterfind('.//script[@type="application/ld+json"]'):
        if elem.text and '"article' in elem.text:
            mymatch = re.search(r'"articlebody":"(.+?)","', elem.text, re.I)
            if mymatch:
                postbody = etree.Element('body')
                elem = etree.Element('p')
                elem.text = trim(mymatch.group(1).replace('\\"', '"'))
                postbody.append(elem)
                return postbody, elem.text, len(elem.text)
    # scrape from article tag
    article_elem = tree.find('.//article')  # |.//main
    if article_elem is not None:  # len(elems) > 0:
        temp_text = trim(article_elem.text_content())
        len_text = len(temp_text)
        if len_text > 0:
            elem = etree.Element('p')
            elem.text = temp_text
            postbody.append(elem)
            return postbody, temp_text, len_text
    # scrape from text paragraphs
    results = set()
    for element in tree.iter('blockquote', 'code', 'p', 'pre', 'q', 'quote'):
        entry = element.text_content()
        if entry not in results:
            elem = etree.Element('p')
            elem.text = entry
            postbody.append(elem)
            results.add(entry)
            # elem.getparent().remove(elem)
    temp_text = trim('\n'.join(postbody.itertext()))
    return postbody, temp_text, len(temp_text)


def determine_returnstring(docmeta, output_format, include_formatting, include_links, tei_validation):
    '''Convert XML tree to chosen format, clean the result and output it as a string'''
    # XML (TEI) steps
    if 'xml' in output_format:
        # last cleaning
        for element in docmeta['body'].iter():
            if element.tag != 'graphic' and len(element) == 0 and not element.text and not element.tail:
                parent = element.getparent()
                if parent is not None:
                    parent.remove(element)
        # build output trees
        if output_format == 'xml':
            output = build_xml_output(docmeta)
        elif output_format == 'xmltei':
            output = build_tei_output(docmeta)
        # can be improved
        returnstring = control_xml_output(
            output, output_format, tei_validation, docmeta)
    # CSV, JSON and TXT output
    else:
        if output_format == 'csv':
            posttext = xmltotxt(
                docmeta['body'], include_formatting, include_links)
            if docmeta['commentsbody'] is not None:
                commentstext = xmltotxt(
                    docmeta['commentsbody'], include_formatting, include_links)
            else:
                commentstext = ''
            returnstring = txttocsv(posttext, commentstext, docmeta)
        elif output_format == 'json':
            returnstring = build_json_output(docmeta)
        else:  # txt
            returnstring = xmltotxt(
                docmeta['body'], include_formatting, include_links)
            if docmeta['commentsbody'] is not None:
                returnstring += '\n' + \
                    xmltotxt(docmeta['commentsbody'],
                             include_formatting, include_links)
                returnstring = returnstring.strip()
    return returnstring


def bare_extraction(filecontent, url=None, no_fallback=False,
                    include_comments=True, output_format='python', target_language=None,
                    include_tables=True, include_images=False, include_formatting=False,
                    include_links=False, deduplicate=False,
                    date_extraction_params=None, with_metadata=False, max_tree_size=None,
                    url_blacklist=None, config=DEFAULT_CONFIG):
    """Internal function for text extraction returning bare Python variables.

    Args:
        filecontent: HTML code as string.
        url: URL of the webpage.
        no_fallback: Skip the backup extraction with readability-lxml and justext.
        include_comments: Extract comments along with the main text.
        output_format: Define an output format, Python being the default
            and the interest of this internal function.
            Other values: 'txt', 'csv', 'json', 'xml', or 'xmltei'.
        target_language: Define a language to discard invalid documents (ISO 639-1 format).
        include_tables: Take into account information within the HTML <table> element.
        include_images: Take images into account (experimental).
        include_formatting: Keep structural elements related to formatting
            (present in XML format, converted to markdown otherwise).
        include_links: Keep links along with their targets (experimental).
        deduplicate: Remove duplicate segments and documents.
        date_extraction_params: Provide extraction parameters to htmldate as dict().
        with_metadata: Only keep documents featuring all essential metadata
            (date, title, url).
        max_tree_size: Discard documents with too many elements.
        url_blacklist: Provide a blacklist of URLs as set() to filter out documents.
        config: Directly provide a configparser configuration.

    Returns:
        A Python dict() containing all the extracted information or None.

    Raises:
        ValueError: Extraction problem.
    """
    # # init
    # if url_blacklist is None:
    #     url_blacklist = set()

    # # load data
    # try:
    #     tree = load_html(filecontent)
    #     if tree is None:
    #         raise ValueError

    #     # HTML lang check
    #     if target_language is not None and check_html_lang(tree, target_language) is False:
    #         raise ValueError

    #     # backup (or not) for further processing
    #     if no_fallback is False:
    #         backup_tree = deepcopy(tree)
    #     else:
    #         backup_tree = None

    #     # extract metadata if necessary
    #     if output_format != 'txt':
    #         docmeta = extract_metadata(tree, url, date_extraction_params)
    #         # cut short if extracted URL in blacklist
    #         if docmeta['url'] in url_blacklist:
    #             raise ValueError
    #         # cut short if core elements are missing
    #         if with_metadata is True and any(
    #                 x is None for x in
    #                 [docmeta['date'], docmeta['title'], docmeta['url']]
    #             ):
    #             raise ValueError
    #     else:
    #         docmeta = dict.fromkeys(METADATA_LIST)

    #     # clean + use LXML cleaner
    #     cleaned_tree = tree_cleaning(tree, include_tables, include_images)

    #     # convert tags, the rest does not work without conversion
    #     cleaned_tree = convert_tags(cleaned_tree, include_formatting, include_tables, include_images, include_links)

    #     # comments first, then remove
    #     if include_comments is True:
    #         commentsbody, temp_comments, len_comments, cleaned_tree = extract_comments(cleaned_tree, deduplicate, config)
    #     else:
    #         commentsbody, temp_comments, len_comments = None, '', 0
    #         #for expr in REMOVE_COMMENTS_XPATH:
    #         #    for subtree in cleaned_tree.xpath(expr):
    #         #        subtree.getparent().remove(subtree)

      # extract content
      postbody, temp_text, len_text, sure_thing = extract_content(
          cleaned_tree, include_tables, include_images, include_links, deduplicate, config)

       # compare if necessary
       if no_fallback is False:
            postbody, temp_text, len_text = compare_extraction(
                tree, backup_tree, url, postbody, temp_text, len_text, target_language, include_formatting, include_links, include_images, config)
            # add baseline as additional fallback
            # or len_text < config.getint('DEFAULT', 'MIN_EXTRACTED_SIZE'):
            if len(postbody) == 0:
                postbody, temp_text, len_text = baseline(filecontent)
        else:
            # rescue: try to use original/dirty tree
            if sure_thing is False and len_text < config.getint('DEFAULT', 'MIN_EXTRACTED_SIZE'):
                postbody, temp_text, len_text = baseline(filecontent)
                LOGGER.debug(
                    'non-clean extracted length: %s (extraction)', len_text)

        # tree size sanity check
        if max_tree_size is not None:
            if len(postbody) > max_tree_size:
                LOGGER.warning('output tree too long: %s', len(postbody))
                etree.strip_tags(postbody, 'hi')
                if len(postbody) > max_tree_size:
                    LOGGER.error(
                        'output tree too long: %s, discarding file', len(postbody))
                    raise ValueError
        # size checks
        if len_comments < config.getint('DEFAULT', 'MIN_EXTRACTED_COMM_SIZE'):
            LOGGER.info('not enough comments %s', url)
        if len_text < config.getint('DEFAULT', 'MIN_OUTPUT_SIZE') and \
           len_comments < config.getint('DEFAULT', 'MIN_OUTPUT_COMM_SIZE'):
            LOGGER.info('text and comments not long enough: %s %s',
                        len_text, len_comments)
            raise ValueError

        # check duplicates at body level
        if deduplicate is True and duplicate_test(postbody, config) is True:
            raise ValueError

        # sanity check on language
        if target_language is not None and \
                language_filter(temp_text, temp_comments, target_language, docmeta) is True:
            raise ValueError

    except ValueError:
        # docmeta['url'] , record_id
        LOGGER.info('discarding data for url: %s', url)
        return None

    # special case: python variables
    if output_format == 'python':
        docmeta['text'] = xmltotxt(postbody, include_formatting, include_links)
        if include_comments is True:
            docmeta['comments'] = xmltotxt(
                commentsbody, include_formatting, include_links)
    else:
        docmeta['raw-text'], docmeta['body'], docmeta['commentsbody'] = temp_text, postbody, commentsbody
    return docmeta


def extract(filecontent, url=None, record_id=None, no_fallback=False,
            include_comments=True, output_format='txt',
            tei_validation=False, target_language=None,
            include_tables=True, include_images=False, include_formatting=False,
            include_links=False, deduplicate=False,
            date_extraction_params=None, with_metadata=False, max_tree_size=None, url_blacklist=None,
            settingsfile=None, config=DEFAULT_CONFIG):
    """Main function exposed by the package:
       Wrapper for text extraction and conversion to chosen output format.

    Args:
        filecontent: HTML code as string.
        url: URL of the webpage.
        record_id: Add an ID to the metadata.
        no_fallback: Skip the backup extraction with readability-lxml and justext.
        include_comments: Extract comments along with the main text.
        output_format: Define an output format:
            'txt', 'csv', 'json', 'xml', or 'xmltei'.
        tei_validation: Validate the XML-TEI output with respect to the TEI standard.
        target_language: Define a language to discard invalid documents (ISO 639-1 format).
        include_tables: Take into account information within the HTML <table> element.
        include_images: Take images into account (experimental).
        include_formatting: Keep structural elements related to formatting
            (only valuable if output_format is set to XML).
        include_links: Keep links along with their targets (experimental).
        deduplicate: Remove duplicate segments and documents.
        date_extraction_params: Provide extraction parameters to htmldate as dict().
        with_metadata: Only keep documents featuring all essential metadata
            (date, title, url).
        max_tree_size: Discard documents with too many elements.
        url_blacklist: Provide a blacklist of URLs as set() to filter out documents.
        settingsfile: Use a configuration file to override the standard settings.
        config: Directly provide a configparser configuration.

    Returns:
        A string in the desired format or None.

    """
    # configuration init
    config = use_config(settingsfile, config)
    if url_blacklist is None:
        url_blacklist = set()
    # extraction
    docmeta = bare_extraction(
        filecontent, url=url, no_fallback=no_fallback,
        include_comments=include_comments, output_format=output_format,
        target_language=target_language, include_tables=include_tables, include_images=include_images,
        include_formatting=include_formatting, include_links=include_links,
        deduplicate=deduplicate,
        date_extraction_params=date_extraction_params, with_metadata=with_metadata,
        max_tree_size=max_tree_size, url_blacklist=url_blacklist, config=config,
    )
    if docmeta is None:
        return None
    if output_format != 'txt':
        # add record ID to metadata
        docmeta['id'] = record_id
        # calculate fingerprint
        docmeta['fingerprint'] = content_fingerprint(docmeta['raw-text'])
    # return
    return determine_returnstring(docmeta, output_format, include_formatting, include_links, tei_validation)


# for legacy and backwards compatibility
process_record = extract
