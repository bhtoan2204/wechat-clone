package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go-socket/core/shared/config"
	"go-socket/core/shared/contracts/events"
	"go-socket/core/shared/pkg/stackErr"

	es8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type elasticsearchMessageIndexer struct {
	client *es8.Client
	index  string
}

func NewElasticsearchMessageIndexer(cfg config.ElasticsearchConfig, client *es8.Client) (events.MessageSearchIndexer, error) {
	if !cfg.Enabled || client == nil {
		return nil, nil
	}

	indexer := &elasticsearchMessageIndexer{
		client: client,
		index:  normalizeIndexName(cfg.RoomMessageIndex),
	}

	if err := indexer.ensureIndex(context.Background()); err != nil {
		return nil, stackErr.Error(err)
	}

	return indexer, nil
}

func (i *elasticsearchMessageIndexer) UpsertMessage(ctx context.Context, document *events.SearchMessageDocument) error {
	if i == nil || i.client == nil || document == nil {
		return nil
	}

	body, err := json.Marshal(document)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal elasticsearch room message failed: %v", err))
	}

	req := esapi.IndexRequest{
		Index:      i.index,
		DocumentID: document.MessageID,
		Body:       bytes.NewReader(body),
		Refresh:    "false",
	}
	res, err := req.Do(ctx, i.client)
	if err != nil {
		return stackErr.Error(fmt.Errorf("index elasticsearch room message failed: %v", err))
	}
	defer res.Body.Close()

	if res.IsError() {
		return stackErr.Error(fmt.Errorf("index elasticsearch room message returned status %s: %s", res.Status(), readBody(res.Body)))
	}
	return nil
}

func (i *elasticsearchMessageIndexer) ensureIndex(ctx context.Context) error {
	existsReq := esapi.IndicesExistsRequest{Index: []string{i.index}}
	existsRes, err := existsReq.Do(ctx, i.client)
	if err != nil {
		return stackErr.Error(fmt.Errorf("check elasticsearch index failed: %v", err))
	}
	defer existsRes.Body.Close()

	switch existsRes.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
	default:
		return stackErr.Error(fmt.Errorf("check elasticsearch index returned status %s: %s", existsRes.Status(), readBody(existsRes.Body)))
	}

	body, err := json.Marshal(roomMessageIndexDefinition())
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal elasticsearch index definition failed: %v", err))
	}

	createReq := esapi.IndicesCreateRequest{
		Index: i.index,
		Body:  bytes.NewReader(body),
	}
	createRes, err := createReq.Do(ctx, i.client)
	if err != nil {
		return stackErr.Error(fmt.Errorf("create elasticsearch index failed: %v", err))
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		payload := readBody(createRes.Body)
		if createRes.StatusCode == http.StatusBadRequest && strings.Contains(payload, "resource_already_exists_exception") {
			return nil
		}
		return stackErr.Error(fmt.Errorf("create elasticsearch index returned status %s: %s", createRes.Status(), payload))
	}

	return nil
}

func roomMessageIndexDefinition() map[string]interface{} {
	return map[string]interface{}{
		"settings": map[string]interface{}{
			"analysis": map[string]interface{}{
				"analyzer": map[string]interface{}{
					"room_message_text": map[string]interface{}{
						"tokenizer": "standard",
						"filter":    []string{"lowercase", "asciifolding"},
					},
				},
			},
		},
		"mappings": map[string]interface{}{
			"dynamic": "strict",
			"properties": map[string]interface{}{
				"room_id": map[string]interface{}{"type": "keyword"},
				"room_name": map[string]interface{}{
					"type":     "text",
					"analyzer": "room_message_text",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{"type": "keyword", "ignore_above": 256},
					},
				},
				"room_type":  map[string]interface{}{"type": "keyword"},
				"message_id": map[string]interface{}{"type": "keyword"},
				"message_content": map[string]interface{}{
					"type":     "text",
					"analyzer": "room_message_text",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{"type": "keyword", "ignore_above": 1024},
					},
				},
				"message_type":              map[string]interface{}{"type": "keyword"},
				"reply_to_message_id":       map[string]interface{}{"type": "keyword"},
				"forwarded_from_message_id": map[string]interface{}{"type": "keyword"},
				"file_name": map[string]interface{}{
					"type":     "text",
					"analyzer": "room_message_text",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{"type": "keyword", "ignore_above": 512},
					},
				},
				"file_size":         map[string]interface{}{"type": "long"},
				"mime_type":         map[string]interface{}{"type": "keyword"},
				"object_key":        map[string]interface{}{"type": "keyword"},
				"message_sender_id": map[string]interface{}{"type": "keyword"},
				"message_sender_name": map[string]interface{}{
					"type":     "text",
					"analyzer": "room_message_text",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{"type": "keyword", "ignore_above": 256},
					},
				},
				"message_sender_email":  map[string]interface{}{"type": "keyword"},
				"message_sent_at":       map[string]interface{}{"type": "date"},
				"mention_all":           map[string]interface{}{"type": "boolean"},
				"mentioned_account_ids": map[string]interface{}{"type": "keyword"},
				"mentions": map[string]interface{}{
					"type": "nested",
					"properties": map[string]interface{}{
						"account_id": map[string]interface{}{"type": "keyword"},
						"display_name": map[string]interface{}{
							"type":     "text",
							"analyzer": "room_message_text",
							"fields": map[string]interface{}{
								"keyword": map[string]interface{}{"type": "keyword", "ignore_above": 256},
							},
						},
						"username": map[string]interface{}{"type": "keyword"},
					},
				},
			},
		},
	}
}

func normalizeIndexName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "room_messages_v1"
	}
	return value
}

func readBody(body io.Reader) string {
	if body == nil {
		return ""
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return ""
	}
	return string(data)
}
