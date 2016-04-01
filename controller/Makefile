EXECUTABLE=../bin/controller

.PHONY: all

all: $(EXECUTABLE)

$(EXECUTABLE): ./main.go ./cfunc.go \
	../block/block.pb.go
	go build -o $(EXECUTABLE)

