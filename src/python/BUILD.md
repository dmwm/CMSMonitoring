# change version in init file
vim CMSMonitoring/__init__.py

# build distribution tar ball
python setup.py sdist

# upload to pypi
twine upload --username ABC -p XXX --verbose dist/*
