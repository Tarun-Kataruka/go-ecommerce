package database

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/Tarun-Kataruka/ecommerce/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrorCantFindProduct    = errors.New("can't find the product")
	ErrorCantDecodeProducts = errors.New("can't find the products")
	ErrorUserIdIsNotValid   = errors.New("this user is not valid")
	ErrorCantUpdateUser     = errors.New("cannot add this product to the cart")
	ErrorCantRemoveItemCart = errors.New("cannot remove this product from the cart")
	ErrorCantGetItem        = errors.New("unable to get the item from the cart")
	ErrorCantBuyCartItem    = errors.New("cannot update the purchase")
)

func AddProductToCart(ctx context.Context, prodCollection, userCollection *mongo.Collection, productID primitive.ObjectID, userID string) error {
	searchfromdb, err := prodCollection.Find(ctx, bson.M{"_id": productID})
	if err != nil {
		log.Println(err)
		return ErrorCantFindProduct
	}
	var productCart []models.ProductUser
	if err = searchfromdb.All(ctx, &productCart); err != nil {
		log.Println(err)
		return ErrorCantDecodeProducts
	}
	user_id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		log.Println(err)
		return ErrorUserIdIsNotValid
	}
	filter := bson.D{primitive.E{Key: "_id", Value: user_id}}
	update := bson.D{{Key: "$push", Value: bson.D{primitive.E{Key: "usercart", Value: bson.D{primitive.E{Key: "$each", Value: productCart}}}}}}
	_, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Println(err)
		return ErrorCantUpdateUser
	}
	return nil
}

func RemoveCartItem(ctx context.Context, prodCollection, userCollection *mongo.Collection, productID primitive.ObjectID, userID string) error {
	user_id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		log.Println(err)
		return ErrorUserIdIsNotValid
	}
	filter := bson.D{primitive.E{Key: "_id", Value: user_id}}
	update := bson.M{"$pull": bson.M{"usercart": bson.M{"_id": productID}}}
	_, err = userCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Println(err)
		return ErrorCantRemoveItemCart
	}
	return nil
}

func BuyItemFromCart(ctx context.Context, userCollection *mongo.Collection, userID string) error {
	//fetch the cart of the user
	//find the cart total
	//create an order with the items
	//empty up the cart
	user_id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		log.Println(err)
		return ErrorUserIdIsNotValid
	}
	var getcartitems models.User
	var ordercart models.Order

	ordercart.Order_ID = primitive.NewObjectID()
	ordercart.Ordered_At = time.Now()
	ordercart.Order_Cart = make([]models.ProductUser, 0)
	ordercart.Payment_Method.COD = true

	unwind := bson.D{{Key: "$unwind", Value: bson.D{primitive.E{Key: "path", Value: "$user_cart"}}}}
	grouping := bson.D{{Key: "$group", Value: bson.D{primitive.E{Key: "_id", Value: "$_id"},
		{Key: "total", Value: bson.D{primitive.E{Key: "$sum", Value: "$user_cart.price"}}}}}}
	currentresults, err := userCollection.Aggregate(ctx, mongo.Pipeline{unwind, grouping})
	ctx.Done()
	if err != nil {
		log.Println(err)
		return ErrorCantGetItem
	}
	var getusercart []bson.M
	if err = currentresults.All(ctx, &getusercart); err != nil {
		panic(err)
	}
	var total_price int32
	for _, user_item := range getusercart {
		price := user_item["total"]
		total_price = price.(int32)
	}
	ordercart.Price = int(total_price)

	filter := bson.D{primitive.E{Key: "_id", Value: user_id}}
	update := bson.D{{Key: "$push", Value: bson.D{primitive.E{Key: "orders", Value: ordercart}}}}
	_, err = userCollection.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Println(err)
		return ErrorCantBuyCartItem
	}
	err = userCollection.FindOne(ctx, bson.D{primitive.E{Key: "_id", Value: user_id}}).Decode(&getcartitems)
	if err != nil {
		log.Println(err)
		return ErrorCantGetItem
	}

	filter2 := bson.D{primitive.E{Key: "_id", Value: user_id}}
	update2 := bson.M{"$push": bson.M{"orders.$[].order_list":bson.M{"$each":getcartitems.UserCart}}}
	_, err = userCollection.UpdateOne(ctx, filter2,update2)
	if err != nil {
		log.Println(err)
	}
	usercart_empty := make([]models.ProductUser, 0)
	update3 := bson.D{primitive.E{Key:"$set", Value: bson.D{primitive.E{Key:"usercart", Value:usercart_empty}}}}
	_, err = userCollection.UpdateOne(ctx, filter2, update3)
	if  err != nil {
		log.Println(err)
		return ErrorCantBuyCartItem
	}
	return nil
}

func InstantBuy(ctx context.Context, prodCollection, userCollection *mongo.Collection, productID primitive.ObjectID, userID string ) error {
	user_id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		log.Println(err)
		return ErrorUserIdIsNotValid
	}

	var product_details models.ProductUser
	var orders_details models.Order

	orders_details.Order_ID = primitive.NewObjectID()
	orders_details.Ordered_At = time.Now()
	orders_details.Order_Cart = make([]models.ProductUser, 0)
	orders_details.Payment_Method.COD = true
	err = prodCollection.FindOne(ctx, bson.D{primitive.E{Key :"_id", Value :productID}}).Decode(&product_details)
	if err != nil {
		log.Println(err)
		return ErrorCantFindProduct
	}
	orders_details.Price = int(*product_details.Price)

	filter := bson.D{primitive.E{Key:"_id", Value: user_id}}
	update := bson.D{{Key: "$push", Value: bson.D{primitive.E{Key: "orders", Value: orders_details}}}}
	_, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Println(err)
	}
	
	filter2 := bson.D{primitive.E{Key: "_id", Value: user_id}}
	update2 := bson.M{"$push" : bson.M{"orders.$[].order_list": product_details}}
	_, err = userCollection.UpdateOne(ctx, filter2, update2)
	if err != nil {
		log.Println(err)
	}
	return nil
}
