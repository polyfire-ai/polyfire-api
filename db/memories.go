package db

import (
	"encoding/json"
	"strconv"
	"strings"
)

type Memory struct {
	ID     string `json:"id"`
	UserId string `json:"user_id"`
	Public bool   `json:"public"`
}

type MatchParams struct {
	QueryEmbedding []float32 `json:"query_embedding"`
	MatchTreshold  float64   `json:"match_threshold"`
	MatchCount     int16     `json:"match_count"`
	MemoryID       []string  `json:"memoryid"`
	UserID         string    `json:"userid"`
}

type MatchResult struct {
	ID         string  `json:"id"`
	Content    string  `json:"content"`
	Similarity float64 `json:"similarity"`
}

type Embedding struct {
	MemoryId  string    `json:"memory_id"`
	UserId    string    `json:"user_id"`
	Content   string    `json:"content"`
	Embedding []float32 `json:"embedding"`
}

func CreateMemory(memoryId string, userId string, public bool) error {
	err := DB.Exec("INSERT INTO memories (id, user_id, public) VALUES (?, ?, ?)", memoryId, userId, public).Error
	if err != nil {
		return err
	}

	return err
}

func AddMemory(userId string, memoryId string, content string, embedding []float32) error {
	embeddingstr := ""
	for _, v := range embedding {
		embeddingstr += strconv.FormatFloat(float64(v), 'f', 6, 64) + ","
	}
	embeddingstr = strings.TrimRight(embeddingstr, ",")

	err := DB.Exec(
		"INSERT INTO embeddings (memory_id, user_id, content, embedding) VALUES (?, ?, ?, string_to_array(?, ',')::float[])",
		memoryId,
		userId,
		content,
		embeddingstr,
	).Error
	if err != nil {
		return err
	}

	return err
}

type MemoryRecord struct {
	ID string `json:"id"`
}

func (MemoryRecord) TableName() string {
	return "memories"
}

func GetMemoryIds(userId string) ([]MemoryRecord, error) {
	var results []MemoryRecord

	err := DB.Find(&results, "user_id = ?", userId).Error
	if err != nil {
		return nil, err
	}

	return results, nil
}

func MatchEmbeddings(memoryIds []string, userId string, embedding []float32) ([]MatchResult, error) {
	params := MatchParams{
		QueryEmbedding: embedding,
		MatchTreshold:  0.70,
		MatchCount:     10,
		MemoryID:       memoryIds,
		UserID:         userId,
	}

	client, err := CreateClient()
	if err != nil {
		return nil, err
	}

	response := client.Rpc("retrieve_embeddings", "", params)

	var results []MatchResult
	err = json.Unmarshal([]byte(response), &results)

	if err != nil {
		return nil, err
	}

	if client.ClientError != nil {
		return nil, err
	}

	return results, nil
}
