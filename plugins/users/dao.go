package users

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/appbaseio-confidential/arc/internal/types/op"
	"github.com/appbaseio-confidential/arc/internal/types/user"
	"github.com/olivere/elastic"
)

type elasticsearch struct {
	url       string
	indexName string
	typeName  string
	mapping   string
	client    *elastic.Client
}

func NewES(url, indexName, typeName, mapping string) (*elasticsearch, error) {
	opts := []elastic.ClientOptionFunc{
		elastic.SetURL(url),
		elastic.SetSniff(false),
	}
	ctx := context.Background()

	// Initialize the client
	client, err := elastic.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("%s: error while initializing elastic client: %v\n", logTag, err)
	}
	es := &elasticsearch{url, indexName, typeName, mapping, client}

	// Check if the meta index already exists
	exists, err := client.IndexExists(indexName).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: error while checking if index already exists: %v\n", logTag, err)
	}
	if exists {
		log.Printf("%s: index named '%s' already exists, skipping...", logTag, indexName)
		return es, nil
	}

	// Meta index does not exists, create a new one
	_, err = client.CreateIndex(indexName).Body(mapping).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: error while creating index named %s: %v\n", logTag, indexName, err)
	}

	log.Printf("%s successfully created index named '%s'", logTag, indexName)
	return es, nil
}

func (es *elasticsearch) getRawUser(userId string) ([]byte, error) {
	data, err := es.client.Get().
		Index(es.indexName).
		Type(es.typeName).
		Id(userId).
		FetchSource(true).
		Do(context.Background())
	if err != nil {
		return nil, err
	}
	src, err := data.Source.MarshalJSON()
	if err != nil {
		return nil, err
	}

	return src, nil
}

func (es *elasticsearch) putUser(u user.User) (bool, error) {
	resp, err := es.client.Index().
		Index(es.indexName).
		Type(es.typeName).
		Id(u.UserId).
		BodyJson(u).
		Do(context.Background())
	if err != nil {
		return false, err
	}

	raw, _ := json.Marshal(resp)
	log.Printf("%s: es_response: %s\n", logTag, string(raw))
	return true, nil
}

func (es *elasticsearch) patchUser(userId string, u user.User) (bool, error) {
	// Only consider fields that can be updated
	fields := make(map[string]interface{})
	if u.ACL != nil && len(u.ACL) >= 0 {
		fields["acl"] = u.ACL
	}
	if u.Email != "" {
		fields["email"] = u.Email
	}
	if u.Op != op.Noop {
		fields["op"] = u.Op
	}
	if u.Indices != nil && len(u.Indices) >= 0 {
		fields["indices"] = u.Indices
	}

	resp, err := es.client.Update().
		Index(es.indexName).
		Type(es.typeName).
		Id(userId).
		Doc(fields).
		Do(context.Background())
	if err != nil {
		return false, err
	}

	raw, _ := json.Marshal(resp)
	log.Printf("%s: es_response: %s\n", logTag, string(raw))
	return true, nil
}

func (es *elasticsearch) deleteUser(userId string) (bool, error) {
	resp, err := es.client.Delete().
		Index(es.indexName).
		Type(es.typeName).
		Id(userId).
		Do(context.Background())
	if err != nil {
		return false, err
	}

	raw, _ := json.Marshal(resp)
	log.Printf("%s: es_response: %s\n", logTag, string(raw))
	return true, nil
}