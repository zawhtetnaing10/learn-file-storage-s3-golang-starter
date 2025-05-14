package main

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"path/filepath"

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

	// Upload thumbnail
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
	mediaType, _, mimeErr := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if mimeErr != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing mine type", mimeErr)
		return
	}
	// if mime type is wrong, respond with an error
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Must upload an image", errors.New("must upload an image"))
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

	// Create the file extension
	fileExtension := strings.TrimPrefix(mediaType, "image/")
	// Crete File Url
	fileUrl := fmt.Sprintf("/assets/%v.%v", videoID, fileExtension)
	// Create file location
	fileLocation := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%v.%v", videoID, fileExtension))
	// Create the file
	newFile, fileErr := os.Create(fileLocation)
	if fileErr != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating a file", fileErr)
		return
	}
	defer newFile.Close()
	// Copy the contents
	io.Copy(newFile, file)

	// Save the url
	video.ThumbnailURL = &fileUrl

	// Update the db
	if vidUpdateErr := cfg.db.UpdateVideo(video); vidUpdateErr != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", vidUpdateErr)
		return
	}

	// Respond with the updated
	respondWithJSON(w, http.StatusOK, video)
}
