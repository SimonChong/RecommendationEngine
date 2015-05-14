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

var movieRatingCount = make(map[int]int)
var movieRatingSum = make(map[int]int)

//UserID -> (MovieID -> Rating)
var userRatingsTraining = make(map[int]map[int]Rating)
var userRatingsValidation = make(map[int]map[int]Rating)

var userRatingCount = make(map[int]int)

type UserCS struct {
	userID           int
	cosineSimilarity float64
}

type UserCSSortable []UserCS

func (this UserCSSortable) Len() int {
	return len(this)
}
func (this UserCSSortable) Less(i, j int) bool {
	return this[i].cosineSimilarity < this[j].cosineSimilarity
}
func (this UserCSSortable) Swap(i, j int) {
	this[i], this[j] = this[j], this[i]
}

var userCSs UserCSSortable
var csPoint = (func() [movies]float64 {
	var rtn [movies]float64
	for i := 0; i < movies; i++ {
		rtn[i] = 1
	}
	return rtn
})()

var userCSIndex = make(map[int]int)

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

	var calcCS = func(userID int) {
		ratings := userRatingsTraining[userID]
		numerator := 0.0
		for _, value := range ratings {
			numerator += float64(value.rating) * maxRating
		}

		sum1 := 0.0
		sum2 := 0.0
		for _, value := range ratings {
			sum1 += math.Pow(float64(value.rating), 2)
			sum2 += maxRating
		}
		denominator := sum1 * sum2
		// fmt.Println(numerator, denominator)

		userCSs = append(userCSs, UserCS{userID, numerator / denominator})
	}

	for userID := range userRatingsTraining {
		calcCS(userID)
	}

	sort.Sort(userCSs)

	for i, cs := range userCSs {
		userCSIndex[cs.userID] = i
	}
}

func findClosestUsers(userID int, topK int, topUsers *UserCSSortable) {
	userIndex := userCSIndex[userID]
	userCS := userCSs[userIndex].cosineSimilarity
	left := userIndex - 1
	right := userIndex + 1
	count := len(userCSs)

	var leftDiff, rightDiff float64
	for i := 0; i < topK && (left >= 0 || right < count); i++ {
		if left >= 0 {
			leftDiff = math.Abs(userCS - userCSs[left].cosineSimilarity)
		} else {
			leftDiff = -1
		}
		if right < len(userCSs) {
			rightDiff = math.Abs(userCS - userCSs[right].cosineSimilarity)
		} else {
			rightDiff = -1
		}
		if leftDiff > rightDiff {
			*topUsers = append(*topUsers, userCSs[left])
			left--
		} else {
			*topUsers = append(*topUsers, userCSs[right])
			right++
		}
	}
}

func predictUserRating(userID int, movieID int, topK int) float64 {
	userIndex := userCSIndex[userID]
	userCS := userCSs[userIndex].cosineSimilarity
	left := userIndex - 1
	right := userIndex + 1
	count := len(userCSs)

	similars := 0      //Users that are closeest and have raited the movie
	similarsSum := 0.0 //Sum of all ratings (to be used for avg calculation)

	var leftDiff, rightDiff float64
	for i := 0; i < count && similars < topK && (left >= 0 || right < count); i++ {
		if left >= 0 {
			leftDiff = math.Abs(userCS - userCSs[left].cosineSimilarity)
		} else {
			leftDiff = -1
		}
		if right < len(userCSs) {
			rightDiff = math.Abs(userCS - userCSs[right].cosineSimilarity)
		} else {
			rightDiff = -1
		}
		var closestUserID int
		//Get the closest User
		if leftDiff > rightDiff {
			closestUserID = userCSs[left].userID
			left--
		} else {
			closestUserID = userCSs[right].userID
			right++
		}

		//Find whether they have rated the movie
		rating, rated := userRatingsTraining[closestUserID][movieID]
		if rated {
			similars++
			similarsSum += float64(rating.rating)
		}
	}
	if similars <= 0 {
		if _, ok := movieRatingCount[movieID]; ok {
			return float64(movieRatingSum[movieID]) / float64(movieRatingCount[movieID])
		} else {
			return 3
		}
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
			// fmt.Printf("UserID: %4d, Prediction vs Actual - Average Difference: %2.2f \n", userID, (float64(predictionDiffSum) / float64(predictionCount)))
		}
		fmt.Println("++++++++++++++++++++++++++++")
		fmt.Printf(" Global Prediction vs Actual | Average Difference: %2.2f | K: %4d\n", (float64(globalPredictionDiffSum) / float64(globalPredictionCount)), k)
		fmt.Println("++++++++++++++++++++++++++++")
	}

}
