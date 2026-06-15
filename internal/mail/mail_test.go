package mail

import (
	"net/textproto"
	"testing"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
)

func TestParseFromAddressUsesAddressForEnvelope(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		wantHeader   string
		wantEnvelope string
	}{
		{
			name:         "plain address",
			raw:          "noreply@example.com",
			wantHeader:   "<noreply@example.com>",
			wantEnvelope: "noreply@example.com",
		},
		{
			name:         "display name",
			raw:          "CUMT Nexus <noreply@example.com>",
			wantHeader:   `"CUMT Nexus" <noreply@example.com>`,
			wantEnvelope: "noreply@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, envelope, err := parseFromAddress(tt.raw)
			if err != nil {
				t.Fatalf("parseFromAddress returned error: %v", err)
			}
			if header != tt.wantHeader {
				t.Fatalf("expected header %q, got %q", tt.wantHeader, header)
			}
			if envelope != tt.wantEnvelope {
				t.Fatalf("expected envelope %q, got %q", tt.wantEnvelope, envelope)
			}
		})
	}
}

func TestParseFromAddressRejectsInvalidFrom(t *testing.T) {
	if _, _, err := parseFromAddress("CUMT Nexus"); err == nil {
		t.Fatal("expected invalid from address to fail")
	}
}

func TestMapSMTPRecipientErrorMapsPermanentReject(t *testing.T) {
	err := mapSMTPRecipientError(&textproto.Error{Code: 550, Msg: "Mailbox not found or access denied"})
	if !apperr.IsCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected invalid_argument, got %v", err)
	}
}

func TestMapSMTPRecipientErrorKeepsTransientErrorInternal(t *testing.T) {
	err := mapSMTPRecipientError(&textproto.Error{Code: 450, Msg: "Mailbox unavailable"})
	if apperr.IsCode(err, apperr.CodeInvalidArgument) {
		t.Fatalf("expected transient error to remain non-business error, got %v", err)
	}
}
