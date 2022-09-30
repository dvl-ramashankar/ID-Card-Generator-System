package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"system/entity"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Connection struct {
	Server     string
	Database   string
	Collection string
}

var Collection *mongo.Collection
var ctx = context.TODO()

func (e *Connection) Connect() {
	clientOptions := options.Client().ApplyURI(e.Server)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	Collection = client.Database(e.Database).Collection(e.Collection)
}

// ===========================Store data & Return card Id======================================
func (e *Connection) CreateIdAndStore(dataBody entity.Request) (string, error) {
	bool, err := validateByNameAndDob(dataBody)
	if err != nil {
		return "", err
	}
	if !bool {
		return "", errors.New("User already present")
	}
	data, err := fetchDataByActive()
	if err != nil {
		return "", err
	}
	var id int64
	fmt.Println("Lowest:", data[0].IdCard)
	fmt.Println("Highest", data[len(data)-1].IdCard)
	if len(data) != 0 {
		id = data[len(data)-1].IdCard + 1
	} else {
		id = 1
	}
	saveData, err := SetValueInModel(dataBody, id)
	if err != nil {
		return "", errors.New("Unable to parse date")
	}
	if _, err := Collection.InsertOne(ctx, saveData); err != nil {
		log.Println(err)
		return "", errors.New("Unable to store data")
	}

	return "Generated Id : " + fmt.Sprintf("%v", id), nil
}

// XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
// ==============================Fetch All Data======================================
func (e *Connection) FetchAllData() ([]*entity.Data, error) {
	var finaldata []*entity.Data
	fetchDataCursor, err := Collection.Find(ctx, bson.D{primitive.E{Key: "active", Value: true}})
	if err != nil {
		return finaldata, err
	}
	finaldata, err = convertDbResultIntoStruct(fetchDataCursor)
	if err != nil {
		return finaldata, err
	}
	if finaldata == nil {
		return finaldata, errors.New("Either Db is empty or all data is deactivated")
	}
	return finaldata, err
}

// ============================Search Data by IdCard================================
func (e *Connection) FetchDataByIdCard(idStr string) ([]*entity.Data, error) {
	var finaldata []*entity.Data
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return finaldata, err
	}
	fetchDataCursor, err := Collection.Find(ctx, bson.D{{"_id", id}, {"active", true}})
	if err != nil {
		return finaldata, err
	}
	fmt.Println(fetchDataCursor)
	finaldata, err = convertDbResultIntoStruct(fetchDataCursor)
	if err != nil {
		return finaldata, err
	}
	fmt.Println(finaldata)
	if len(finaldata) == 0 {
		return finaldata, errors.New("Data not present in db given by Id or it is deactivated")
	}

	return finaldata, err
}

//XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
//=======================================UpdateData by Id=============================

func (e *Connection) UpdateDataById(idStr string, reqData entity.Request) (bson.M, error) {
	var updatedDocument bson.M
	idk, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return updatedDocument, err
	}

	filter := bson.D{
		{"$and",
			bson.A{
				bson.D{{"_id", idk}},
				bson.D{{"active", true}},
			},
		},
	}
	UpdateQuery := bson.D{}
	if reqData.Name != "" {
		UpdateQuery = append(UpdateQuery, primitive.E{Key: "name", Value: reqData.Name})
	}
	if reqData.Age != 0 {
		UpdateQuery = append(UpdateQuery, primitive.E{Key: "age", Value: reqData.Age})
	}
	if reqData.BloodGroup != "" {
		UpdateQuery = append(UpdateQuery, primitive.E{Key: "blood_group", Value: reqData.BloodGroup})
	}
	if reqData.Designation != "" {
		UpdateQuery = append(UpdateQuery, primitive.E{Key: "designation", Value: reqData.Designation})
	}
	if reqData.DOB != "" {
		dob, err := convertDate(reqData.DOB)
		if err != nil {
			log.Println(err)
			return updatedDocument, err
		}
		UpdateQuery = append(UpdateQuery, primitive.E{Key: "dob", Value: dob})
	}
	if reqData.JoiningDate != "" {
		joiningDate, err := convertDate(reqData.JoiningDate)
		if err != nil {
			log.Println(err)
			return updatedDocument, err
		}
		UpdateQuery = append(UpdateQuery, primitive.E{Key: "dob", Value: joiningDate})
	}
	update := bson.D{{"$set", UpdateQuery}}

	r := Collection.FindOneAndUpdate(ctx, filter, update).Decode(&updatedDocument)
	if r != nil {
		return updatedDocument, r
	}
	fmt.Println(updatedDocument)
	if updatedDocument == nil {
		return updatedDocument, errors.New("Data not present in db given by Id or it is deactivated")
	}
	return updatedDocument, err
}

// XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
// ==============================Deactivate document By Id ============================
func (e *Connection) DeleteById(idStr string) (string, error) {
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return "", err
	}
	filter := bson.D{primitive.E{Key: "_id", Value: id}}
	update := bson.D{{"$set", bson.D{primitive.E{Key: "active", Value: false}}}}
	Collection.FindOneAndUpdate(ctx, filter, update)
	return "Documents Deactivated Successfully", err
}

//XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX

func SetValueInModel(req entity.Request, id int64) (entity.Data, error) {
	var data entity.Data
	joiningDate, err := convertDate(req.JoiningDate)
	if err != nil {
		log.Println(err)
		return data, err
	}
	dob, err := convertDate(req.DOB)
	if err != nil {
		log.Println(err)
		return data, err
	}
	data.JoiningDate = joiningDate
	data.DOB = dob
	data.CreatedDate = time.Now()
	data.Name = req.Name
	data.Age = req.Age
	data.Designation = req.Designation
	data.BloodGroup = req.BloodGroup
	data.Active = true
	data.IdCard = id
	return data, nil
}

func fetchDataByActive() ([]*entity.Data, error) {
	var result []*entity.Data
	filter := bson.D{}
	sorting := options.Find().SetSort(bson.D{{"id", -1}})
	data, err := Collection.Find(ctx, filter, sorting)
	if err != nil {
		return result, err
	}
	result, err = convertDbResultIntoStruct(data)
	if err != nil {
		return result, err
	}

	return result, err
}

func convertDate(dateStr string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		log.Println(err)
		return date, err
	}
	return date, nil
}

func validateByNameAndDob(reqbody entity.Request) (bool, error) {
	dobStr := reqbody.DOB
	dob, err := convertDate(dobStr)
	if err != nil {
		return false, err
	}
	fmt.Println(dob)
	var result []*entity.Data
	data, err := Collection.Find(ctx, bson.D{{"name", reqbody.Name}, {"dob", dob}, {"active", true}})
	if err != nil {
		return false, err
	}
	result, err = convertDbResultIntoStruct(data)
	if err != nil {
		return false, err
	}
	if len(result) == 0 {
		return true, err
	}
	return false, err
}

func convertDbResultIntoStruct(fetchDataCursor *mongo.Cursor) ([]*entity.Data, error) {
	var finaldata []*entity.Data
	for fetchDataCursor.Next(ctx) {
		var data entity.Data
		err := fetchDataCursor.Decode(&data)
		if err != nil {
			return finaldata, err
		}
		finaldata = append(finaldata, &data)
	}
	return finaldata, nil
}
