package handlers

import "errors"

var errInvalidWebhookURL = errors.New("invalid webhook_url, expected valid http(s) URL")
