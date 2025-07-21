CREATE TABLE IF NOT EXISTS uploads (
  merkle_root   TEXT        PRIMARY KEY,
  file_name     TEXT        NOT NULL,
  file_size     BIGINT      NOT NULL,
  chunk_size    INT         NOT NULL,
  total_chunks  INT         NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS upload_chunks (
  merkle_root   TEXT        NOT NULL REFERENCES uploads(merkle_root) ON DELETE CASCADE,
  chunk_index   INT         NOT NULL,
  leaf_hash     TEXT        NOT NULL,
  uploaded_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (merkle_root, chunk_index)
);


-- Fast lookup of all chunks received for a given file (by its Merkle root)
CREATE INDEX IF NOT EXISTS idx_upload_chunks_by_root
  ON upload_chunks (merkle_root);

-- Fast pruning of old completed uploads
CREATE INDEX IF NOT EXISTS idx_uploads_by_completed_at
  ON uploads (completed_at);
