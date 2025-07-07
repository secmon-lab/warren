package message

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/utils/safe"
)

type SNS struct {
	Type             string `json:"Type"`
	MessageId        string `json:"MessageId"`
	Token            string `json:"Token"`
	TopicArn         string `json:"TopicArn"`
	Subject          string `json:"Subject,omitempty"`
	Message          string `json:"Message"`
	Timestamp        string `json:"Timestamp"`
	SignatureVersion string `json:"SignatureVersion"`
	Signature        string `json:"Signature"`
	SigningCertURL   string `json:"SigningCertURL"`
	SubscribeURL     string `json:"SubscribeURL"`
}

type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

func (x *SNS) Verify(ctx context.Context, client HTTPClient) error {
	parsedURL, err := url.Parse(x.SigningCertURL)
	if err != nil {
		return goerr.Wrap(err, "failed to parse signing cert URL", goerr.T(errs.TagInvalidRequest), goerr.V("url", x.SigningCertURL))
	}

	// Check if the URL is from AWS SNS
	if !strings.HasPrefix(parsedURL.Host, "sns.") || !strings.HasSuffix(parsedURL.Host, ".amazonaws.com") || !strings.HasPrefix(parsedURL.Path, "/SimpleNotificationService-") {
		return goerr.New("invalid signing cert URL", goerr.T(errs.TagInvalidRequest), goerr.V("url", x.SigningCertURL))
	}

	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Get(x.SigningCertURL)
	if err != nil {
		return goerr.Wrap(err, "failed to get signing cert")
	}
	defer safe.Close(ctx, resp.Body)

	certPEM, err := io.ReadAll(resp.Body)
	if err != nil {
		return goerr.Wrap(err, "failed to read cert")
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return goerr.New("failed to decode PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return goerr.Wrap(err, "failed to parse certificate")
	}

	rsaPublicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return goerr.New("certificate does not contain an RSA public key")
	}

	// Build the message string to verify according to AWS SNS spec
	var stringToSign strings.Builder

	// Common fields for all message types
	stringToSign.WriteString("Message\n")
	stringToSign.WriteString(x.Message + "\n")
	stringToSign.WriteString("MessageId\n")
	stringToSign.WriteString(x.MessageId + "\n")

	// Optional Subject field
	if x.Subject != "" {
		stringToSign.WriteString("Subject\n")
		stringToSign.WriteString(x.Subject + "\n")
	}

	// Type-specific fields
	if x.Type == "SubscriptionConfirmation" || x.Type == "UnsubscribeConfirmation" {
		stringToSign.WriteString("SubscribeURL\n")
		stringToSign.WriteString(x.SubscribeURL + "\n")
		stringToSign.WriteString("Token\n")
		stringToSign.WriteString(x.Token + "\n")
	}

	// Common fields for all message types
	stringToSign.WriteString("Timestamp\n")
	stringToSign.WriteString(x.Timestamp + "\n")
	stringToSign.WriteString("TopicArn\n")
	stringToSign.WriteString(x.TopicArn + "\n")
	stringToSign.WriteString("Type\n")
	stringToSign.WriteString(x.Type + "\n")

	signature, err := base64.StdEncoding.DecodeString(x.Signature)
	if err != nil {
		return goerr.Wrap(err, "failed to decode signature")
	}

	var alg x509.SignatureAlgorithm
	var hash crypto.Hash
	switch x.SignatureVersion {
	case "1":
		alg = x509.SHA1WithRSA
		hash = crypto.SHA1
	case "2":
		alg = x509.SHA256WithRSA
		hash = crypto.SHA256
	default:
		return goerr.New("invalid signature version", goerr.T(errs.TagInvalidRequest), goerr.V("version", x.SignatureVersion))
	}

	if err := cert.CheckSignature(alg, []byte(stringToSign.String()), signature); err != nil {
		return goerr.Wrap(err, "signature verification failed")
	}

	hashed := hash.New()
	hashed.Write([]byte(stringToSign.String()))
	digest := hashed.Sum(nil)

	if err := rsa.VerifyPKCS1v15(rsaPublicKey, hash, digest, signature); err != nil {
		return goerr.Wrap(err, "signature verification failed")
	}

	return nil
}
