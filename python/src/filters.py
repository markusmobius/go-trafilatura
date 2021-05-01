"""
Functions related to content filtering, mostly duplicate detection and language
detection.
"""

import base64
import hashlib
import logging
import re

# language detection
try:
    import cld3
    LANGID_FLAG = True
except ImportError:
    LANGID_FLAG = False

from .lru import LRUCache
from .settings import LRU_SIZE
from .utils import trim


LOGGER = logging.getLogger(__name__)

LRU_TEST = LRUCache(maxsize=LRU_SIZE)

RE_HTML_LANG = re.compile(r'([a-z]{2})', re.I)

# COMMENTS_BLACKLIST = ('( Abmelden / Ändern )') # Fill in your details below|Trage deine Daten unten|Kommentar verfassen|Bitte logge dich|Hinterlasse einen Kommentar| to %s| mit %s)


def language_filter(temp_text, temp_comments, target_language, docmeta):
    '''Run external component (if installed) for language identification'''
    # sanity check on language
    if target_language is not None:
        if LANGID_FLAG is True:
            # comments
            if len(temp_comments) > len(temp_text):
                langtest = temp_comments
            # default
            else:
                langtest = temp_text
            result = cld3.get_language(langtest)
            if result.language != target_language:
                LOGGER.warning('wrong language: %s %s %s',
                               result, docmeta['id'], docmeta['url'])
                return True
        else:
            LOGGER.warning('Detector not installed, no language detection run')
    return False


def content_fingerprint(string):
    '''Calculate a hash value for meaningful bits of the content'''
    teststring = ' '.join(re.findall(r'\w{5,}', string.lower()))
    m = hashlib.sha1()
    m.update(teststring.encode())
    fingerprint = m.digest()
    return base64.b64encode(fingerprint).decode()
