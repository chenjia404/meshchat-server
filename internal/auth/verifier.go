package auth

import (
	"encoding/base64"
	"errors"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Verifier validates that a public key maps to the claimed peer_id and signed the challenge.
type Verifier interface {
	Verify(peerID, publicKeyBase64, challenge, signatureBase64 string) error
}

// LibP2PVerifier performs real libp2p public key and signature validation.
type LibP2PVerifier struct{}

func NewLibP2PVerifier() *LibP2PVerifier {
	return &LibP2PVerifier{}
}

func (v *LibP2PVerifier) Verify(peerID, publicKeyBase64, challenge, signatureBase64 string) error {
	if peerID == "" || publicKeyBase64 == "" || signatureBase64 == "" {
		return errors.New("peer_id, public_key and signature are required")
	}

	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return err
	}

	publicKey, err := libp2pcrypto.UnmarshalPublicKey(publicKeyBytes)
	if err != nil {
		return err
	}

	resolvedPeerID, err := peer.IDFromPublicKey(publicKey)
	if err != nil {
		return err
	}
	if resolvedPeerID.String() != peerID {
		return errors.New("public key does not match peer_id")
	}

	signatureBytes, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return err
	}

	ok, err := publicKey.Verify([]byte(challenge), signatureBytes)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("invalid challenge signature")
	}

	return nil
}
