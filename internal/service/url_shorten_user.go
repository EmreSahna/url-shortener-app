package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/EmreSahna/url-shortener-app/internal/hash"
	"github.com/EmreSahna/url-shortener-app/internal/models"
	"github.com/EmreSahna/url-shortener-app/internal/sqlc"
	"github.com/EmreSahna/url-shortener-app/internal/validator"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func (s *service) UrlShortenUser(ctx context.Context, req models.ShortenURLRequest) (res models.ShortenURLResponse, err error) {
	// Validate url
	if !validator.ValidateURL(req.OriginalUrl) {
		return models.ShortenURLResponse{}, models.UrlNotValidErr()
	}

	// Parse token from string
	t := ctx.Value(models.TokenKey).(string)
	c, err := s.jwt.Parse(t)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return models.ShortenURLResponse{}, models.TokenExpiredErr()
		} else if err != nil {
			return models.ShortenURLResponse{}, models.TokenFailureErr()
		}
	}

	// Parse user id from token
	userUUID, err := uuid.Parse(c["id"].(string))
	if err != nil {
		return models.ShortenURLResponse{}, models.CustomerIdParseErr()
	}

	// Validate duration
	var duration time.Duration
	if req.ExpireTime != "" {
		parsedDate, err := validator.ParseDateWithTimeZone(req.ExpireTime)
		if err != nil {
			return models.ShortenURLResponse{}, err
		}

		duration, err = validator.ValidateFutureDate(parsedDate)
		if err != nil {
			return models.ShortenURLResponse{}, err
		}
	}

	// Shorten url
	shortenUrl := hash.GenerateUniqueCode()

	// Save to redis
	redisCh := make(chan error, 1)
	go func() {
		log.Printf("Starting to save shortened URL %s to Redis...", shortenUrl)
		err = s.rcc.SetUrlWithExpire(ctx, shortenUrl, req.OriginalUrl, duration)
		if err != nil {
			redisCh <- models.SaveToCacheErr()
		}
		redisCh <- err
	}()

	// Save to db
	dbCh := make(chan error, 1)
	go func() {
		log.Printf("Starting to save shortened URL %s to PostgreSQL...", shortenUrl)
		savedUrl, err := s.db.CreateURL(ctx, sqlc.CreateURLParams{
			OriginalUrl:   req.OriginalUrl,
			ShortenedCode: shortenUrl,
			UserID:        &userUUID,
		})
		if err != nil {
			dbCh <- models.CreateURLErr()
		}

		err = s.db.InsertClickCount(context.Background(), savedUrl.ID)
		if err != nil {
			dbCh <- models.CreateClickCountErr()
		}
		dbCh <- nil
	}()

	if err = <-dbCh; err != nil {
		return models.ShortenURLResponse{}, err
	}
	log.Printf("Successfully saved shortened URL %s to Redis.", shortenUrl)

	if err = <-redisCh; err != nil {
		return models.ShortenURLResponse{}, err
	}
	log.Printf("Successfully saved shortened URL %s to PostgreSQL.", shortenUrl)

	// Return
	return models.ShortenURLResponse{
		Url: shortenUrl,
	}, nil
}
