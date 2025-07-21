package model

type InitUploadRequest struct {
    MerkleRoot  string `json:"merkle_root"`  // client-computed
    FileName    string `json:"file_name"`
    FileSize    int64  `json:"file_size"`
    ChunkSize   int    `json:"chunk_size"`
    TotalChunks int    `json:"total_chunks"`
}

type InitUploadResponse struct {
    MerkleRoot  string `json:"merkle_root"`
    TotalChunks int    `json:"total_chunks"`
    Uploaded    []int  `json:"uploaded"`
}