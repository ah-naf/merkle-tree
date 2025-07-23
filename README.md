# Merkle‑Chunk File Transfer

[Demo](https://youtu.be/juZQKpriHrM)

## 🚀 What is this project?

A Go‑based file transfer service that lets you reliably **upload**, **verify**, and **download** large files in **fixed‑size chunks**, using a **Merkle tree** to guarantee integrity at every step.

## 🔍 What does it do?

1. **Chunked Upload**

   - Splits any file into N fixed‑size pieces.
   - Stores each chunk on the server, along with its cryptographic hash.

2. **Integrity Verification**

   - Builds a Merkle tree over all chunk hashes.
   - Exposes a `/verify/:root` endpoint so clients can confirm that **all** chunks are present and unmodified before download.

3. **Chunked Download & Merge**
   - Provides a `/download/:root` endpoint that streams chunks in order.
   - Client CLI fetches, merges, and writes the complete file, with a live progress bar.

## ⚙️ How does it work?

### Server (Gin‑based API)

- **UploadHandler**

  1. Receives chunk uploads (`POST /upload/:root/:index`)
  2. Stores chunk bytes on disk at `storagePath/<merkle_root>/<index>.chunk`
  3. Records `leaf_hash` for each chunk in the database

- **Verification**

  - `GET /verify/:root`
  - Recomputes Merkle root from stored leaf hashes
  - Returns HTTP 200 only if the recomputed root matches the one in the uploads table

- **Download**
  - `GET /download/:root`
  - Streams all chunks in order (0 → N−1) with `Content-Disposition: attachment; filename="<root>.bin"`

### Client (Cobra‑powered CLI)

- **Commands**

  - `upload` – chunk, hash, and push your file to the server
  - `download` – verify integrity, then pull & merge chunks
  - `files` – list all uploaded files

- **Progress Feedback**

  - Uses [schollz/progressbar/v3](https://github.com/schollz/progressbar) for real‑time byte‑count progress.

- **Error Handling**
  - `SilenceUsage: true` so only your custom error message is shown
  - Pretty‑printed JSON for server error details

---

## 📦 Installation & Usage

1. **Server**

   ```bash
   cd server/
   go build -o merkle-server ./cmd/server

   ```

2. **Client**

   ```bash
   cd cli/
   go build -o merkle ./cmd/merkle
   ./merkle upload --root <hash> --file mylarge.bin
   ./merkle files --server http://localhost:8080
   ./merkle download --root <hash> --out mylarge.bin
   ```
