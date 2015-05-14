# Recommendation Engine
Recommendation Engine for Data Science Workshop

## Approach
- Load all the data into memory (its only 30MB)
- Split the data into 2. 
	- 2/3 rds of the data will be used for training
	- 1/3 rd of the data will be used to verify our predictions
- Calculate the Cosine Similarity between user ratings relative to each other user
- Sort all user Consine Similarity numbers
- Predict movie ratings using the values calculated
	- Predict a movie rating by finding the closest K users that have already rated the movie.
	- Then average the ratings of the closest K users that have rated the movie.

	
###Results

- To calculate the effectiveness of our predictions we use the validation data set.
	- For each movie in the validation set we calculate a prediction
	- From that prediction we calculate the difference between the actual and predicted
	- Averaging the difference across all movies and user gives us a view of how accurate it is.
- We can tweak the K value (number of closest users) to make it more or less accurate.

