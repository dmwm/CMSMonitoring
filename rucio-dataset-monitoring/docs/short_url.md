### Short URL implementation

This service creates short url using incoming request's MD5 hash and MongoDB collection to store hash and request match

In Short:

- Create a MD5 hash from incoming request
- Store hash id - request couple in MongoDB collection
- When a user provide hash id, return a page which is restored from the matching request and state with hash id in
  MongoDB

How to short url works:

- When user clicked the "Copy state link to clipboard" button, DataTables sends request to short_url API
- You can see incoming request structure: models.ShortUrlRequest
- This JSOn request converted to JSON string and an MD5 hash is created from this string in `getRequestHash` function
- This hash id is checked if there is an already matching hash id in MongoDB `short_url` collection
- If not, `getShortUrl` function inserts models.ShortUrl to MongoDB.
- models.ShortUrl contains all we need to restore the DataTables page state: shared query and shared state(HTML/CSS)
- After insertion, hash id can be shared safely with other users.

How to use short url:

- User can use shared hash id in `short-url/:id` endpoint
- Controller fetches the models.ShortUrl from the given id
- Fetched DataTable request and state used in `index.html` to restore the state if the page to shared hash id request
- Mainly Go template helps to modify JavaScript variables/objects and HTML

