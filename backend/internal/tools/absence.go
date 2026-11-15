package tools

import (
	"aibuddy/internal/clients/hr"
	"aibuddy/internal/domain"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type absenceArgs struct {
	Type     string `json:"type"`
	DateFrom string `json:"date_from"`
	DateTo   string `json:"date_to"`
	Comment  string `json:"comment"`
}

type CreateAbsence struct {
	HR hr.Client
}

func (CreateAbsence) Name() string {return "create_absence_request"}
func (CreateAbsence) Description() string {
	return "Создать заявку на отгул или отпуск для текущего пользователя"
}
func (CreateAbsence) RequiresConfirmation() bool {return true}
func (CreateAbsence) Schema() json.RawMessage {
	return json.RawMessage(`{
	"type": "object",
	"required": ["type", "date_from", "date_to"],
	"additionalProperties": false,
	"properties": {
	  "type": {"type": "string", "enum": ["day_off", "vacation"]},
	  "date_from": {"type": "string", "format": "date"},
	  "date_to": {"type": "string", "format": "date"},
	  "comment": {"type": "string", "maxLength": 300}
	  }
	}`)
}

func parseAbsence(raw json.RawMessage) (absenceArgs, error) {
	var a absenceArgs
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&a); err != nil {
		return a, err
	}
	return a, nil
}

func (CreateAbsence) Validate(raw json.RawMessage) error {
	a, err := parseAbsence(raw)
	if err != nil {
		return err
	}
	if a.Type != "day_off" && a.Type != "vacation" {
		return fmt.Errorf("type must be day_off or vacation")
	}
	from, err := time.Parse("2006-01-02", a.DateFrom)
	if err != nil {
		return fmt.Errorf("date_from: %w", err)
	}
	to, err := time.Parse("2006-01-02", a.DateTo)
	if err != nil {
		return fmt.Errorf("date_to: %w", err)
	}
	if to.Before(from) {
		return errors.New("date_to before date_from")
	}
	if len(a.Comment) > 300 {
		return errors.New("comment too long")
	}
	return nil
}

func (c CreateAbsence) Execute(ctx context.Context, id domain.Identity, raw json.RawMessage, idemKey string) (Output, error) {
	a, err := parseAbsence(raw)
	if err != nil {
		return Output{}, err
	}
	res, err := c.HR.CreateAbsence(ctx, hr.CreateAbsenceReq{
		UserID: id.UserID, Type: a.Type, DateFrom: a.DateFrom, DateTo: a.DateTo, Comment: a.Comment,
	}, idemKey)
	if err != nil {
		return Output{}, err
	}
	data, err := json.Marshal(res)
	if err != nil {
		return Output{}, err
	}
	return Output{Data: data, ExternalID: res.ID}, nil
}