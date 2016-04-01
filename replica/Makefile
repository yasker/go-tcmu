EXECUTABLE=../bin/replica

.PHONY: all

all: $(EXECUTABLE)

$(EXECUTABLE): ./main.go \
	../block/block.pb.go
	go build -o $(EXECUTABLE)

