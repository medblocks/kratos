package oidc

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/tidwall/sjson"

	"github.com/ory/x/decoderx"
)

func decoderRegistration(ref string) (decoderx.HTTPDecoderOption, error) {
	raw, err := sjson.SetBytes([]byte(registrationFormPayloadSchema), "properties.traits.$ref", ref)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	o, err := decoderx.HTTPRawJSONSchemaCompiler(raw)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return o, nil
}

// merge merges the userFormValues (extracted from the initial POST request) prefixed with `traits` (encoded) with the
// values coming from the OpenID Provider (openIDProviderValues).
func merge(userFormValues string, openIDProviderValues json.RawMessage, option decoderx.HTTPDecoderOption) (json.RawMessage, error) {
	if userFormValues == "" {
		return openIDProviderValues, nil
	}

	var decodedForm struct {
		Traits map[string]interface{} `json:"traits"`
	}

	req, err := http.NewRequest("POST", "/", bytes.NewBufferString(userFormValues))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if err := decoderx.NewHTTP().Decode(
		req, &decodedForm,
		decoderx.HTTPFormDecoder(),
		option,
		decoderx.HTTPDecoderSetIgnoreParseErrorsStrategy(decoderx.ParseErrorIgnore),
		decoderx.HTTPDecoderSetValidatePayloads(false),
	); err != nil {
		return nil, err
	}

	var decodedTraits map[string]interface{}
	if err := json.NewDecoder(bytes.NewBuffer(openIDProviderValues)).Decode(&decodedTraits); err != nil {
		return nil, err
	}

	// decoderForm (coming from POST request) overrides decodedTraits (coming from OP)
	if err := mergo.Merge(&decodedTraits, decodedForm.Traits, mergo.WithOverride); err != nil {
		return nil, err
	}

	var result bytes.Buffer
	if err := json.NewEncoder(&result).Encode(decodedTraits); err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
