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

# COMMENTS_BLACKLIST = ('( Abmelden / Ändern )') # Fill in your details below|Trage deine Daten unten|Kommentar verfassen|Bitte logge dich|Hinterlasse einen Kommentar| to %s| mit %s)


def content_fingerprint(string):
    '''Calculate a hash value for meaningful bits of the content'''
    teststring = ' '.join(re.findall(r'\w{5,}', string.lower()))
    m = hashlib.sha1()
    m.update(teststring.encode())
    fingerprint = m.digest()
    return base64.b64encode(fingerprint).decode()
