package domain

import "go.mongodb.org/mongo-driver/bson/primitive"

type Product struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Quantity    uint64             `bson:"quantity" json:"quantity"`
	Description string             `bson:"description" json:"description"`
	Price       float64            `bson:"price" json:"price"`
}

type ProductImage struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	ProductID primitive.ObjectID `bson:"product_id"`
	ImageURL  string             `bson:"image_url"`
}
