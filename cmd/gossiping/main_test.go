package main

import (
	"encoding/json"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUnmarshalTargetgroup(t *testing.T) {
	var tg targetgroup.Group
	var text = []byte(`
{
        "labels": {
                "foo": "bar"
        },
        "targets": [
                "172.17.210.182"
        ]
}

`)

	err := json.Unmarshal(text, &tg)
	require.NoError(t, err)
}
