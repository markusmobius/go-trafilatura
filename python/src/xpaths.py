"""
X-Path expressions needed to extract and filter the main text content
"""
# This file is available from https://github.com/adbar/trafilatura
# under GNU GPL v3 license


DISCARD_XPATH = [
    '''.//*[contains(@id, "footer") or contains(@class, "footer") or
    contains(@id, "bottom") or contains(@class, "bottom")]''',
    # related posts, sharing jp-post-flair jp-relatedposts, news outlets + navigation
    # or self::article
    '''.//*[(self::div or self::item or self::list or self::p or self::section or self::span)][
    contains(@id, "related") or contains(translate(@class, "R","r"), "related") or
    contains(@id, "viral") or contains(@class, "viral") or
    starts-with(@id, "shar") or starts-with(@class, "shar") or contains(@class, "share-") or
    contains(@id, "social") or contains(@class, "social") or contains(@class, "sociable") or
    contains(@id, "syndication") or contains(@class, "syndication") or
    starts-with(@id, "jp-") or starts-with(@id, "dpsp-content")
    or
    contains(@id, "teaser") or contains(translate(@class, "T","t"), "teaser")
    or contains(@id, "newsletter") or contains(@class, "newsletter") or
    contains(@id, "cookie") or contains(@class, "cookie") or contains(@id, "tags")
    or contains(@class, "tags")  or contains(@id, "sidebar") or
    contains(@class, "sidebar") or contains(@id, "banner") or contains(@class, "banner")
    or contains(@class, "meta") or
    contains(@id, "menu") or contains(@class, "menu") or
    starts-with(@id, "nav") or starts-with(@class, "nav") or
    contains(@id, "navigation") or contains(translate(@class, "N","n"), "navigation")
    or contains(@role, "navigation")
    or contains(@class, "navbox") or starts-with(@class, "post-nav")
    or contains(@id, "breadcrumb") or contains(@class, "breadcrumb") or
    contains(@id, "bread-crumb") or contains(@class, "bread-crumb") or
    contains(@id, "author") or contains(@class, "author") or
    contains(@id, "button") or contains(@class, "button")
    or contains(@id, "caption") or contains(@class, "caption") or
    contains(translate(@class, "B","b"), "byline")
    or contains(@class, "rating") or starts-with(@class, "widget") or
    contains(@class, "attachment") or contains(@class, "timestamp") or
    contains(@class, "user-info") or contains(@class, "user-profile") or
    contains(@class, "-ad-") or contains(@class, "-icon")
    or contains(@class, "article-infos") or
    contains(translate(@class, "I","i"), "infoline")]''',
    # comment debris
    '''.//*[@class="comments-title" or contains(@class, "comments-title") or contains(@class, "nocomments") or starts-with(@id, "reply-") or starts-with(@class, "reply-") or
    contains(@class, "-reply-") or contains(@class, "message") or contains(@id, "akismet") or contains(@class, "akismet")]''',
    # hidden
    '''.//*[starts-with(@class, "hide-") or contains(@class, "hide-print") or contains(@id, "hidden")
    or contains(@style, "hidden") or contains(@hidden, "hidden") or contains(@class, "noprint") or contains(@style, "display:none") or contains(@class, " hidden")]''',
]
# conflicts:
# .//header # contains(@id, "header") or contains(@class, "header") or
# contains(@id, "link") or contains(@class, "link")
# class contains cats
# or contains(@class, "hidden ")
