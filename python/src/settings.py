# pylint:disable-msg=E0611
"""
Listing a series of settings that are applied module-wide.
"""

# This file is available from https://github.com/adbar/trafilatura
# under GNU GPL v3 license

import configparser

from os import cpu_count
from pathlib import Path


def use_config(filename=None, config=None):
    'Use configuration object or read and parse a settings file'
    # expert option: use config file directly
    if config is not None:
        return config
    # default filename
    if filename is None:
        filename = str(Path(__file__).parent / 'settings.cfg')
    # load
    config = configparser.ConfigParser()
    config.read(filename)
    return config


DEFAULT_CONFIG = use_config()


# Safety checks
DOWNLOAD_THREADS = min(cpu_count(), 16)  # 16 processes at most
TIMEOUT = 30
LRU_SIZE = 4096

# Files
MAX_FILES_PER_DIRECTORY = 1000
FILENAME_LEN = 8
FILE_PROCESSING_CORES = min(cpu_count(), 16)  # 16 processes at most

# Network
MAX_SITEMAPS_SEEN = 10000


# filters


JUSTEXT_LANGUAGES = {
    'ar': 'Arabic',
    'bg': 'Bulgarian',
    'cz': 'Czech',
    'da': 'Danish',
    'de': 'German',
    'en': 'English',
    'el': 'Greek',
    'es': 'Spanish',
    'fa': 'Persian',
    'fi': 'Finnish',
    'fr': 'French',
    'hr': 'Croatian',
    'hu': 'Hungarian',
    # 'ja': '',
    'ko': 'Korean',
    'id': 'Indonesian',
    'it': 'Italian',
    'no': 'Norwegian_Nynorsk',
    'nl': 'Dutch',
    'pl': 'Polish',
    'pt': 'Portuguese',
    'ro': 'Romanian',
    'ru': 'Russian',
    'sk': 'Slovak',
    'sl': 'Slovenian',
    'sr': 'Serbian',
    'sv': 'Swedish',
    'tr': 'Turkish',
    'uk': 'Ukranian',
    'ur': 'Urdu',
    'vi': 'Vietnamese',
    # 'zh': '',
}
