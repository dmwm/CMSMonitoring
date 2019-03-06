# tag new release
git tag ...
git push --tags

# build distribution tar ball
python setup.py sdist

# upload to pypi
twine upload --username ABC -p XXX --verbose dist/*
