package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	defer file.Close()

	// Get the media type
	mediaType := header.Header.Get("Content-Type")

	mediaType, _, err = mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "No mime type could be parsed from the Content-Type header", nil)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Only jpeg or png allowed", nil)
		return
	}

	mediaTypeSplit := strings.Split(mediaType, "/")
	fileExtension := mediaTypeSplit[len(mediaTypeSplit)-1]

	// Read the data
	// data, err := io.ReadAll(file)
	// if err != nil {
	// 	respondWithError(w, http.StatusBadRequest, "Couldn't read data", err)
	// 	return
	// }

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

	// Save the media to disk
	filenameByteArray := make([]byte, 32)
	rand.Read(filenameByteArray)
	filenameString := base64.RawURLEncoding.EncodeToString(filenameByteArray)

	thumbnailFilePath := filepath.Join(cfg.assetsRoot, filenameString)
	thumbnailFilePath = fmt.Sprintf("%s.%s", thumbnailFilePath, fileExtension)
	osFile, err := os.Create(thumbnailFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create file", err)
		return
	}

	_, err = io.Copy(osFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy data to file", err)
		return
	}

	// Construct the new thumbnail Url
	thumbnailUrl := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, filenameString, fileExtension)

	// Set the Thumbnail URL
	dbVideo.ThumbnailURL = &thumbnailUrl
	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video in DB", err)
		return
	}

	respondWithJSON(w, http.StatusOK, dbVideo)
}
