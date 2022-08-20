CFLAGS=`pkg-config --cflags libavformat libavutil sdl2 libswresample`
LIBS=`pkg-config --libs libavformat libavutil sdl2 libswresample`
STD=-std=c++0x
DEBUG=-g

SRC_DIR :=.
OBJ_DIR :=.
SRC_FILES := $(wildcard $(SRC_DIR)/*.cpp)
OBJ_FILES := $(patsubst $(SRC_DIR)/%.cpp, $(OBJ_DIR)/%.o, $(SRC_FILES))


%.o:%.cpp
	echo $(CFLAGS)
	g++ $(CFLAGS) $(DEBUG) $(LIBS) $(STD) -c -o $@ $<

hplay:$(OBJ_FILES)
	g++ $(CFLAGS) $(DEBUG) $(LIBS) $(STD) -o $@ $^

all:hplay

clean:
	rm -rf main hplay *.o