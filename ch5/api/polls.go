package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type poll struct {
	ID      primitive.ObjectID `bson:"_id" json:"id"`
	Title   string             `json:"title"`
	Options []string           `json:"options"`
	Results map[string]int     `json:"results,omitempty"`
}

func handlePolls(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		handlePollsGet(w, r)
		return
	case "POST":
		handlePollsPost(w, r)
		return
	case "DELETE":
		handlePollsDelete(w, r)
		return
	case "OPTIONS":
		w.Header().Add("Access-Control-Allow-Methods", "DELETE")
		respond(w, r, http.StatusOK, nil)
		return

	}
	respondHTTPErr(w, r, http.StatusNotFound)
}

func handlePollsGet(w http.ResponseWriter, r *http.Request) {
	db := GetVar(r, "db").(*mongo.Database)
	c := db.Collection("polls")

	var q *mongo.Cursor
	var objID primitive.ObjectID

	p := NewPath(r.URL.Path)

	if p.HasID() {
		objID, _ = primitive.ObjectIDFromHex(p.ID)
		q, _ = c.Find(context.Background(), bson.M{"_id": objID})
	} else {
		q, _ = c.Find(context.Background(), bson.D{{}})
	}

	var result []*poll
	if err := q.All(context.Background(), &result); err != nil {
		respondErr(w, r, http.StatusInternalServerError, err)
		return
	}
	fmt.Println(&result)
}

func handlePollsPost(w http.ResponseWriter, r *http.Request) {
	ctx, err := context.WithTimeout(context.Background(), 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	db := GetVar(r, "db").(*mongo.Database)
	c := db.Collection("polls")
	var p poll
	if err := decodeBody(r, &p); err != nil {
		respondErr(w, r, http.StatusBadRequest, "リクエストから調査項目を読み込めません", err)
		return
	}
	p.ID = primitive.NewObjectID()
	if _, err := c.InsertOne(ctx, p); err != nil {
		respondErr(w, r, http.StatusInternalServerError, "調査項目の格納に失敗しました", err)
		return
	}
	w.Header().Set("Location", "polls/"+p.ID.Hex())
	respond(w, r, http.StatusCreated, nil)
}

func handlePollsDelete(w http.ResponseWriter, r *http.Request) {
	ctx, err := context.WithTimeout(context.Background(), 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	db := GetVar(r, "db").(*mongo.Database)
	c := db.Collection("polls")
	p := NewPath(r.URL.Path)

	if !p.HasID() {
		respondErr(w, r, http.StatusMethodNotAllowed, "すべての調査項目を削除することはできません")
		return
	}

	objID, _ := primitive.ObjectIDFromHex(p.ID)
	if _, err := c.DeleteOne(ctx, objID); err != nil {
		respondErr(w, r, http.StatusInternalServerError, "調査項目の削除に失敗しました", err)
		return
	}
	respond(w, r, http.StatusOK, nil)
}
