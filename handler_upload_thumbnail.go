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
	const maxMemory = 10 << 20

	// Parse the form data
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Form", err)
		return
	}

	// Get the image data
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get file", err)
		return
	}

	// Get the media type
	mediaType := header.Header.Get("Content-Type")

	// Read the data
	data, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't read data", err)
		return
	}

	// Get the video metadata from the DB
	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get video metadata from DB", err)
		return
	}

	// Check if video belongs to user
	if dbVideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Video doesn't belong to users", err)
		return
	}

	// Create a new Thumbnail to store
	videoThumbnail := thumbnail{
		data:      data,
		mediaType: mediaType,
	}

	// Store the thumbnail in the global map
	videoThumbnails[videoID] = videoThumbnail

	// Set the Thumbnail URL
	url := fmt.Sprintf("http://localhost:%s/api/thumbnails/%s", cfg.port, videoIDString)
	dbVideo.ThumbnailURL = &url
	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video in DB", err)
		return
	}

	respondWithJSON(w, http.StatusOK, dbVideo)
}
