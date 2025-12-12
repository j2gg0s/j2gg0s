from transformers import CLIPTokenizer

folder = "/Volumes/jdisk/huggingface/hub/models--black-forest-labs--FLUX.1-dev/snapshots/0ef5fff789c832c5c7f4e127f94c8b54bbcced44"

tokenizer = CLIPTokenizer.from_pretrained(folder + "/tokenizer")

prompt = "Hello World!"
print(tokenizer._tokenize(prompt))
print(tokenizer(prompt))
