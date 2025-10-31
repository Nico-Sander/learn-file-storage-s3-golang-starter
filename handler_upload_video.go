package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// Set an upload limit of 1GB, by wrapping r.Body and replacing it with the wrapper
	readCloser := http.MaxBytesReader(w, r.Body, 1<<30)
	r.Body = readCloser

	// Extract the videoID from the URL path parameters and parse it as a UUID
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Video ID", err)
		return
	}

	// Authenticate the user and get a userID
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

	// Get the video metadata from the DB, Check if the authenticated user is the owner
	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 404, "Video with given ID does not exist in DB", err)
		return
	}

	if dbVideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Requested video doesn't belong to authenticated user.", nil)
	}

	// Parse the uploaded video file from the form data
	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get FormFile", err)
		return
	}
	defer file.Close()

	// Validate the uploaded file to ensure it's mp4
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse media type", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Only video files are allowed", nil)
	}

	// Safe the file to a temporary file on disk
	tmpFileName := "tubely-upload.mp4"
	tmpFile, err := os.CreateTemp("", tmpFileName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create temporary file", err)
		return
	}

	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy data to temporary file", err)
		return
	}

	// Reset the temp file's pointer to the beginning
	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't reset the temporary file's pointer", err)
		return
	}

	// Get the aspect ratio of the video
	aspectRatio, err := getVideoAspectRatio(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't determine aspect ratio", err)
	}

	// Create a prefix depending on the aspect ratio
	aspectRatioPrefix := "other/"
	if aspectRatio == "16:9" {
		aspectRatioPrefix = "landscape/"
	} else if aspectRatio == "9:16" {
		aspectRatioPrefix = "portrait/"
	}

	// Generate a random filename
	filenameByteArray := make([]byte, 32)
	rand.Read(filenameByteArray)
	filenameString := base64.RawURLEncoding.EncodeToString(filenameByteArray)

	// Get the file extension
	mediaTypeSplit := strings.Split(mediaType, "/")
	fileExtension := mediaTypeSplit[len(mediaTypeSplit)-1]

	// Construct the entire filename
	fullFileName := fmt.Sprintf("%s%s.%s", aspectRatioPrefix, filenameString, fileExtension)

	// Put the object into S3
	cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fullFileName,
		Body:        tmpFile,
		ContentType: &mediaType,
	})

	// Update the VideoURL in the DB
	videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, fullFileName)
	dbVideo.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video URL in DB", err)
		return
	}

	// Respond with the new video
	respondWithJSON(w, http.StatusOK, dbVideo)

}
