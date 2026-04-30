# logstore

A simple HTTP server for file storage and retrieval.

**Features:**
- POST `/` to upload a file to the server, storing it with under a random directory
- POST `/` a form to upload contents of a file, using `contents` and `filename` fields
- POST `/{random_directory}` to upload a file to the server, storing it with under a previously created directory or "session"
- GET `/logs/{random_directory}/{filename}` to retrieve files
- GET `/d/{random_directory}/tar` to retrieve a tar file with contents of the "session" 
- GET `/logs/` to get a list of available directories
- GET `/` for web UI (HTML or plain text), if it is from a browser, an html form will be displayed
- Max upload size: 32MB
- Runs on port 3000
- Temporary working directory that auto-cleans on shutdown

No authentication or security features are implemented, functional features only.

Use at your own risk. Not intended for long-term or production use, code quality is questionable.