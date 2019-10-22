import os

extensions = ['sphinx.ext.autodoc',
    'sphinx.ext.doctest',
    'sphinx.ext.todo',
    'sphinx.ext.viewcode']

doctest_path = [os.path.dirname(os.path.dirname(__file__))]
