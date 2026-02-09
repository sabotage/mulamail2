package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ---------- client ----------

type Client struct {
	client *mongo.Client
	db     *mongo.Database
}

func Connect(uri, dbName string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return &Client{client: client, db: client.Database(dbName)}, nil
}

func (c *Client) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.client.Disconnect(ctx)
}

// ---------- models ----------

// Identity maps an email address to a Solana public key, optionally anchored
// by a on-chain memo transaction.
type Identity struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email     string             `bson:"email"        json:"email"`
	PubKey    string             `bson:"pubkey"       json:"pubkey"`
	TxHash    string             `bson:"tx_hash"      json:"tx_hash,omitempty"`
	Verified  bool               `bson:"verified"     json:"verified"`
	CreatedAt time.Time          `bson:"created_at"   json:"created_at"`
}

// MailAccount stores connection details for one legacy mail server.
// Passwords are encrypted at rest; the PassEnc fields are never serialised
// back to the client (json:"-").
type MailAccount struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OwnerPubKey  string             `bson:"owner_pubkey"  json:"owner_pubkey"`
	AccountEmail string             `bson:"account_email" json:"account_email"`
	POP3         POP3Settings       `bson:"pop3"          json:"pop3"`
	SMTP         SMTPSettings       `bson:"smtp"          json:"smtp"`
	CreatedAt    time.Time          `bson:"created_at"    json:"created_at"`
}

type POP3Settings struct {
	Host    string `bson:"host"     json:"host"`
	Port    int    `bson:"port"     json:"port"`
	User    string `bson:"user"     json:"user"`
	PassEnc string `bson:"pass_enc" json:"-"`
	UseSSL  bool   `bson:"use_ssl"  json:"use_ssl"`
}

type SMTPSettings struct {
	Host    string `bson:"host"     json:"host"`
	Port    int    `bson:"port"     json:"port"`
	User    string `bson:"user"     json:"user"`
	PassEnc string `bson:"pass_enc" json:"-"`
	UseSSL  bool   `bson:"use_ssl"  json:"use_ssl"`
}

// ---------- identity operations ----------

func (c *Client) CreateIdentity(ctx context.Context, id *Identity) error {
	id.CreatedAt = time.Now()
	result, err := c.db.Collection("identities").InsertOne(ctx, id)
	if err != nil {
		return err
	}
	// Set the ID from the insert result
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		id.ID = oid
	}
	return nil
}

func (c *Client) GetIdentityByEmail(ctx context.Context, email string) (*Identity, error) {
	var id Identity
	err := c.db.Collection("identities").FindOne(ctx, bson.M{"email": email}).Decode(&id)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (c *Client) GetIdentityByPubKey(ctx context.Context, pubkey string) (*Identity, error) {
	var id Identity
	err := c.db.Collection("identities").FindOne(ctx, bson.M{"pubkey": pubkey}).Decode(&id)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// ---------- mail-account operations ----------

func (c *Client) CreateMailAccount(ctx context.Context, acc *MailAccount) error {
	acc.CreatedAt = time.Now()
	result, err := c.db.Collection("mail_accounts").InsertOne(ctx, acc)
	if err != nil {
		return err
	}
	// Set the ID from the insert result
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		acc.ID = oid
	}
	return nil
}

func (c *Client) GetMailAccountsByOwner(ctx context.Context, ownerPubKey string) ([]MailAccount, error) {
	cursor, err := c.db.Collection("mail_accounts").Find(ctx, bson.M{"owner_pubkey": ownerPubKey})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	accounts := make([]MailAccount, 0)
	if err := cursor.All(ctx, &accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}

func (c *Client) GetMailAccount(ctx context.Context, ownerPubKey, accountEmail string) (*MailAccount, error) {
	var acc MailAccount
	err := c.db.Collection("mail_accounts").FindOne(ctx, bson.M{
		"owner_pubkey":  ownerPubKey,
		"account_email": accountEmail,
	}).Decode(&acc)
	if err != nil {
		return nil, err
	}
	return &acc, nil
}
