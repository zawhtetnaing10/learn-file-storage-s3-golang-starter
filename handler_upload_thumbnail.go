package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	// Set up max memory
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	// Get the thumbnail file
	file, header, fileErr := r.FormFile("thumbnail")
	if fileErr != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", fileErr)
		return
	}
	defer file.Close()

	// Get Media Type
	mediaType := header.Header.Get("Content-Type")
	// Read the file
	fileBytes, readErr := io.ReadAll(file)
	if readErr != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to read the file", readErr)
		return
	}

	// Get Video Meta Data
	video, getVidErr := cfg.db.GetVideo(videoID)
	if getVidErr != nil {
		respondWithError(w, http.StatusNotFound, "Video not found", getVidErr)
		return
	}

	// Check video's user id
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "you don't have permission to upload the thumbnail", getVidErr)
		return
	}

	// Create thumbnail struct
	tn := thumbnail{
		data:      fileBytes,
		mediaType: mediaType,
	}
	// Add to the videoThumbnails map
	videoThumbnails[videoID] = tn

	// Set thumbnail url to video and update the db
	thumbnailUrl := fmt.Sprintf("http://localhost:8091/api/thumbnails/%v", videoID)
	video.ThumbnailURL = &thumbnailUrl
	if vidUpdateErr := cfg.db.UpdateVideo(video); vidUpdateErr != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", vidUpdateErr)
		return
	}

	// Respond with the updated
	respondWithJSON(w, http.StatusOK, video)
}
