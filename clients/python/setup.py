try:
    from setuptools import setup
    setup  # quiet "redefinition of unused ..." warning from pyflakes
    # arguments that distutils doesn't understand
    setuptools_kwargs = {
        'install_requires': [
        ],
        'provides': ['queued'],
        'zip_safe': False
        }
except ImportError:
    from distutils.core import setup
    setuptools_kwargs = {}

setup(name='queued',
      version=1.1,
      description=(
        'A client for queued'
      ),
      author='Tim Henderson',
      author_email='tadh@case.edu',
      url='queued.org',
      packages=['queued',],
      platforms=['unix'],
      scripts=[],
      **setuptools_kwargs
)

