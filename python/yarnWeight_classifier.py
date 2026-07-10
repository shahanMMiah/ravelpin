import matplotlib.pyplot as plt
import locale
import pathlib

import tensorflow as tf

from tensorflow import keras
from tensorflow.keras import layers
from tensorflow.keras.models import Sequential
from tensorflow.keras.optimizers import Adam


def main():
    batch_size = 32
    img_height = 224
    img_width = 224
    epochs = 20

    
    datset_fp = "./assets/datasets/yarnweights" 
    pb_fp = "./model/yn_classify"
    data_dir = pathlib.Path(datset_fp)

  
    train_ds = tf.keras.utils.image_dataset_from_directory(
    data_dir,
    validation_split=0.2,
    subset="training",
    seed=123,
    image_size=(img_height, img_width),
    batch_size=batch_size)

    val_ds = tf.keras.utils.image_dataset_from_directory(
    data_dir,
    validation_split=0.2,
    subset="validation",
    seed=123,
    image_size=(img_height, img_width),
    batch_size=batch_size)
    
    early_stopping = tf.keras.callbacks.EarlyStopping(
    monitor='val_loss',    
    patience=6,             
    restore_best_weights=True
)


    class_names = train_ds.class_names
    print(class_names)
    
    with open(pathlib.Path(pb_fp, "labels.txt"), mode="w", encoding= locale.getpreferredencoding(False)) as obj:
        for nme in  class_names:
            obj.write(f"{nme}\n")

    autotune = tf.data.AUTOTUNE

    train_ds = train_ds.cache().shuffle(1000).prefetch(buffer_size=autotune)
    val_ds = val_ds.cache().prefetch(buffer_size=autotune)

 
    num_classes = len(class_names)

    data_augmentation = keras.Sequential(
    [
        layers.RandomFlip("horizontal",
                        input_shape=(img_height,
                                    img_width,
                                    3)),
        layers.RandomRotation(0.1),
        layers.RandomContrast(0.1)
    ]
    )
    leaky_relu_alpha = 0.1

    model = Sequential([
    data_augmentation,
    layers.Rescaling(1./255,  input_shape=(img_height, img_width, 3)),
    layers.Conv2D(16, 3, padding='same'),
    layers.LeakyReLU(alpha=leaky_relu_alpha),
    layers.MaxPooling2D(),

    layers.Conv2D(32, 3, padding='same'),
    layers.LeakyReLU(alpha=leaky_relu_alpha),
    layers.MaxPooling2D(),

    layers.Conv2D(64, 3, padding='same'),
    layers.LeakyReLU(alpha=leaky_relu_alpha),
    layers.MaxPooling2D(),

    layers.Flatten(),
    layers.Dropout(0.5),

    layers.Dense(128),
    
    layers.LeakyReLU(alpha=leaky_relu_alpha),
    layers.Dropout(0.5),

    layers.Dense(num_classes, name="outputs")
    ])

    model.compile(optimizer=Adam(),
              loss=tf.keras.losses.SparseCategoricalCrossentropy(from_logits=True),
              metrics=['accuracy'])


    history = model.fit(
    train_ds,
    validation_data=val_ds,
    epochs=epochs,
    #callbacks=[early_stopping]
    )

    model.save(pb_fp)


if __name__ == "__main__":
    print("yarn classify!")
    main()
   