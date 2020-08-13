package bot

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ConfigId string

type Command struct {
	Text      string
	TimeAfter int64
}

type MessageConfig struct {
	Id         ConfigId   `bson:"_id, omitempty"`
	SpecConfig SpecConfig `bson:"config"`
}

type Repo interface {
	Get(ctx context.Context, id ConfigId) (*MessageConfig, error)
	Save(ctx context.Context, userState *MessageConfig) error
	List(ctx context.Context) ([]MessageConfig, error)
}

type messageConfigRepo struct {
	storage    map[ConfigId]MessageConfig
	collection mongo.Collection
}

func (b messageConfigRepo) Get(ctx context.Context, id ConfigId) (*MessageConfig, error) {
	_, err := primitive.ObjectIDFromHex(string(id))
	if err != nil {
		return nil, err
	}

	var result MessageConfig

	if err := b.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (b messageConfigRepo) Save(ctx context.Context, messageConfig *MessageConfig) error {
	messageConfig.Id = ConfigId(primitive.NewObjectID().Hex())
	_, err := b.collection.InsertOne(ctx, messageConfig)

	return err
}

func (b messageConfigRepo) List(ctx context.Context) ([]MessageConfig, error) {
	result := []MessageConfig{}

	cur, err := b.collection.Find(ctx, bson.D{{}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.TODO())

	// Finding multiple documents returns a cursor
	// Iterating through the cursor allows us to decode documents one at a time
	for cur.Next(context.TODO()) {

		// create a value into which the single document can be decoded
		var elem MessageConfig
		if err := cur.Decode(&elem); err != nil {
			return nil, err
		}

		result = append(result, elem)
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	// Close the cursor once finished
	return result, nil
}

func NewConfigRepo(db *mongo.Database) Repo {
	coll := db.Collection("sync_config")

	return messageConfigRepo{
		collection: *coll,
	}
}
