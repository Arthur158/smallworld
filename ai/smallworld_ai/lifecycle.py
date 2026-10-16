"""Bound disk and memory growth for continuous training."""
import logging
from logging.handlers import RotatingFileHandler
import re


class JsonlLog:
    def __init__(self, path, max_bytes=64 << 20, backups=2):
        self.handler = RotatingFileHandler(path, maxBytes=max_bytes, backupCount=backups, encoding='utf-8')
        self.handler.setFormatter(logging.Formatter('%(message)s'))

    def write(self, line):
        self.handler.emit(logging.LogRecord('training', logging.INFO, '', 0, line.rstrip('\n'), (), None))

    def __enter__(self):
        return self

    def __exit__(self, *_):
        self.handler.close()


def prune_checkpoints(directory, keep):
    """Retain recent numbered checkpoints; never touch parents, exports or latest."""
    if keep < 0:
        raise ValueError('checkpoint retention must be non-negative')
    if not keep:
        return
    checkpoints = []
    for path in directory.glob('step-*.pt'):
        match = re.fullmatch(r'step-(\d+)\.pt', path.name)
        if match and path.is_file() and not path.is_symlink():
            checkpoints.append((int(match[1]), path))
    for _, path in sorted(checkpoints, reverse=True)[keep:]:
        path.unlink()
