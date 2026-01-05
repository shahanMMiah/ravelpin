curl -L -o ./model/model.tar.gz\
  https://www.kaggle.com/api/v1/models/google/mobilenet-v2/tensorFlow2/100-224-classification/2/download

tar -xvzf ./model/model.tar.gz -C ./model
tar -xvzf filename.tar.gz -C /path/to/directory