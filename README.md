# Reverser

1. docker build -t reverser .
2. docker run --rm -p 8000:8000 rev
3. Visit localhost:8000 :)

The application provides the following functionality
====================================================

1. Can register new proxy url targets ex https://www.google.com, https://www.facebook.com etc with a given identifier such as google, fb
2. Can visit the registered proxy via http://localhost:8000/proxy/google where google is the identifier

Example usage:

1. Register https://golang.org/ as test
2. Visit http://localhost:8000/proxy/test/pkg/net/http/ to see the contents of the target path


