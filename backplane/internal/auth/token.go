package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
)

var (
	DefaultTokenPath = filepath.Join(os.Getenv("HOME"), ".traintrack", "credentials.json")
)

type StoredToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	IDToken      string    `json:"id_token"`
	Expiry       time.Time `json:"expiry"`
}

func SaveToken(path string, tok *oauth2.Token) error {
	idToken, _ := tok.Extra("id_token").(string)

	stored := StoredToken{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		IDToken:      idToken,
		Expiry:       tok.Expiry,
	}

	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func LoadToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var stored StoredToken
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken:  stored.AccessToken,
		RefreshToken: stored.RefreshToken,
		Expiry:       stored.Expiry,
		TokenType:    "Bearer",
	}, nil
}
