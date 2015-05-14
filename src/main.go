package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"sort"
)

type Rating struct {
	rating    int
	timestamp int
}

const movies int = 3952
const maxRating float64 = 5.0

//UserID -> (MovieID -> Rating)
var userRatingsTraining = make(map[int]map[int]Rating)
var userRatingsValidation = make(map[int]map[int]Rating)

var userRatingCount = make(map[int]int)

type UserCS struct {
	userID           int
	cosineSimilarity float64
}

type UserCSSortable []UserCS

func (self UserCSSortable) Len() int {
	return len(self)
}
func (self UserCSSortable) Less(i, j int) bool {
	return self[i].cosineSimilarity < self[j].cosineSimilarity
}
func (self UserCSSortable) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

type CSStore struct {
	cosineSimilarities      UserCSSortable
	cosineSimilaritiesIndex map[int]int //userID -> index of cosine similarity
}

func (self CSStore) Sort() {
	sort.Sort(sort.Reverse(self.cosineSimilarities))
	for i, cs := range self.cosineSimilarities {
		self.cosineSimilaritiesIndex[cs.userID] = i
	}
}

var userCSStore = make(map[int]CSStore) //userID -> cosine similarities

var movieRatingCount = make(map[int]int)
var movieRatingSum = make(map[int]int)

func loadRatings() {
	file, err := os.Open("../data/ratings.txt")
	if err != nil {
		fmt.Println("ratings.txt File not found")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		var userID int
		var movieID int
		var rating Rating
		fmt.Sscanf(line, "%d::%d::%d::%d", &userID, &movieID, &rating.rating, &rating.timestamp)
		count, ok := userRatingCount[userID]
		if !ok {
			count = 1
		} else {
			count++
		}
		if count%3 == 0 {
			_, ok := userRatingsValidation[userID]
			if !ok {
				userRatingsValidation[userID] = make(map[int]Rating)
			}
			userRatingsValidation[userID][movieID] = rating
		} else {
			_, ok := userRatingsTraining[userID]
			if !ok {
				userRatingsTraining[userID] = make(map[int]Rating)
			}
			userRatingsTraining[userID][movieID] = rating
		}
		userRatingCount[userID] = count
		// fmt.Println(userID, count)

		if _, ok := movieRatingCount[movieID]; !ok {
			movieRatingSum[movieID] = 0
			movieRatingCount[movieID] = 0
		}
		movieRatingSum[movieID] += rating.rating
		movieRatingCount[movieID]++

	}
}

func calcUserCS() {
	var appendCS = func(userID0 int, userID1 int, cs float64) {

		if _, ok := userCSStore[userID0]; !ok {
			userCSStore[userID0] = CSStore{make(UserCSSortable, 0), make(map[int]int)}
		}
		if _, ok := userCSStore[userID1]; !ok {
			userCSStore[userID1] = CSStore{make(UserCSSortable, 0), make(map[int]int)}
		}

		csU0 := UserCS{userID1, cs}
		csStore0 := userCSStore[userID0]
		csStore0.cosineSimilaritiesIndex[userID1] = len(csStore0.cosineSimilarities)
		csStore0.cosineSimilarities = append(csStore0.cosineSimilarities, csU0)
		userCSStore[userID0] = csStore0

		// fmt.Println(userCSStore[userID0])
		// fmt.Println(csStore0.cosineSimilarities)

		csU1 := UserCS{userID0, cs}
		csStore1 := userCSStore[userID1]
		csStore1.cosineSimilaritiesIndex[userID0] = len(csStore1.cosineSimilarities)
		csStore1.cosineSimilarities = append(csStore1.cosineSimilarities, csU1)
		userCSStore[userID1] = csStore1

	}

	var calcCS = func(userID0 int, userID1 int) {

		_, hasU := userCSStore[userID0]
		_, hasC := userCSStore[userID0].cosineSimilaritiesIndex[userID1]
		if hasC && hasU {
			return //Already calculated this
		}

		ratings0 := userRatingsTraining[userID0]
		ratings1 := userRatingsTraining[userID1]

		sum1 := 0.0
		sum2 := 0.0

		numerator := 0.0
		for movieID, value0 := range ratings0 {
			value1, ok := ratings1[movieID]
			if ok {
				numerator += float64(value0.rating) * float64(value1.rating)
			}
			sum1 += math.Pow(float64(value0.rating), 2)
			sum2 += math.Pow(float64(value1.rating), 2)
		}
		denominator := sum1 * sum2
		var result float64

		// fmt.Println(numerator, denominator)
		if denominator == 0.0 || numerator == 0.0 {
			if _, ok := movieRatingCount[movieID]; ok {
				result = float64(movieRatingSum[movieID]) / float64(movieRatingCount[movieID])
			} else {
				return 3
			}
		} else {
			result = numerator / denominator
		}

		appendCS(userID0, userID1, result)
	}
	var calcCSMulti = func(userID int) {
		for userIDt, _ := range userRatingsTraining {
			calcCS(userID, userIDt)
		}
		userCSStore[userID].Sort()
	}

	for userID := range userRatingsTraining {
		// fmt.Printf("Calculating CS for user: %4d \n", userID)
		userCSStore[userID] = CSStore{make(UserCSSortable, 0), make(map[int]int)}
		calcCSMulti(userID)
		// fmt.Println(userCSStore[userID])

	}
}

func predictUserRating(userID int, movieID int, topK int) float64 {

	similars := 0      //Users that are closeest and have raited the movie
	similarsSum := 0.0 //Sum of all ratings (to be used for avg calculation)

	for _, cs := range userCSStore[userID].cosineSimilarities {
		if rating, ok := userRatingsTraining[cs.userID][movieID]; ok {
			similars++
			similarsSum += float64(rating.rating)
		}
		if similars >= topK {
			break
		}
	}
	if similarsSum < 1 {
		return 0.0
	}

	return similarsSum / float64(similars)
}

func main() {
	var k = 10
	fmt.Println("Loading Ratings from text file")
	loadRatings()
	fmt.Println("Calculating Cosine Similarities for each user")
	calcUserCS()

	fmt.Println("----------------------------------------------------------")
	fmt.Println(" User Rating Prediction")
	fmt.Println("----------------------------------------------------------")
	var roundRating = func(r float64) int {
		if r < 0 {
			return int(math.Ceil(r - 0.5))
		}
		return int(math.Floor(r + 0.5))
	}
	// fmt.Println(userCSIndex)

	for k = 10; k <= 200; k += 10 {

		globalPredictionCount := 0
		globalPredictionDiffSum := 0

		for userID := range userRatingsValidation {
			predictionCount := 0
			predictionDiffSum := 0
			for movieID, rating := range userRatingsValidation[userID] {
				prediction := predictUserRating(userID, movieID, k)
				predictionRounded := roundRating(prediction)
				// fmt.Printf("UserID: %4d, MovieID: %4d, Prediction: %1.1f , Rounded Prediction: %d, Actual: %d \n",
				// 	userID, movieID, prediction, predictionRounded, rating.rating)

				ratingDiff := rating.rating - predictionRounded
				if ratingDiff < 0 {
					ratingDiff = -ratingDiff
				}
				predictionDiffSum += ratingDiff
				predictionCount++
				globalPredictionDiffSum += ratingDiff
				globalPredictionCount++

			}
			//fmt.Printf("UserID: %4d, Prediction vs Actual - Average Difference: %2.2f \n", userID, (float64(predictionDiffSum) / float64(predictionCount)))
		}
		fmt.Println("++++++++++++++++++++++++++++")
		fmt.Printf(" Global Prediction vs Actual | Average Difference: %2.2f | K = %3d\n", (float64(globalPredictionDiffSum) / float64(globalPredictionCount)), k)
		fmt.Println("++++++++++++++++++++++++++++")
	}
}
