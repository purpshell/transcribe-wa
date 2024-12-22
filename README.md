# transcribe-wa - A Transcription Project
This is one of my playground projects which solves a huge issue for me (as a known voice message spammer).
I also included a bonus feature of mentioning every person in the group chat.

# Installation
Simply clone the git repo to your local device, install Go, then run `go mod tidy`.

# Usage
Run `go build` to generate an executable for your system architecture.
Copy `.env.example` to `.env` and enter your OpenAI key and database settings (only SQLite3 and PostgresSQL are supported).  
Then, run the app (`./transcribe-wa`). Scan the QR code in the settings of the WhatsApp app.

There are the following commands (that only you can run):
- tenable — Enable transcription for the chat
- tdisable — Disable transcription for the chat
- ping — Check if the app is alive
- @everyone — Mentions every single person in the group

# Contact
All my contact options can be found on my [GitHub profile](https://github.com/purpshell).

# Sponsor
If you found this useful, and you'd like to sponsor open-source work like this, please check out my [GitHub Sponsors](https://github.com/sponsors/purpshell) page.

# License
This project is distributed under the "MIT License".

Copyright 2024 Rajeh Taher

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
