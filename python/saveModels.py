import tensorflow as tf
from transformers import TFBertForTokenClassification

# 1. Load model (ensure from_pt=True if loading from a PyTorch checkpoint)
#model = tf.keras.models.load_model("dslim/bert-base-NER")
model = TFBertForTokenClassification.from_pretrained("dslim/bert-base-NER", from_pt=True)
# 2. Define a dummy input to trace the graph (Required for some versions)
# BERT uses input_ids and attention_mask
input_spec = {
    "input_ids": tf.TensorSpec([None, 512], tf.int32, name="input_ids"),
    "attention_mask": tf.TensorSpec([None, 512], tf.int32, name="attention_mask")
}

# 3. Use tf.saved_model.save to force .pb generation
#tf.saved_model.save(model, "./model/bert_slim_ner", signatures=model.call().get_concrete_function(input_spec))
model.save("./model/bert_slim_ner")