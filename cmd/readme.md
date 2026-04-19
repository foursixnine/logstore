# logstore

A simple HTTP server for file storage and retrieval.

**Features:**
- POST files to upload (stores with random filename)
- GET `/logs/{filename}` to retrieve files
- GET `/` for web UI (HTML or plain text)
- Max upload size: 32MB
- Runs on port 3000
- Temporary working directory that auto-cleans on shutdown

No authentication or security features—functional only.

Use at your own risk. Not intended for long-term use, code quality is questionable.