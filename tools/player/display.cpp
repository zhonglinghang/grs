#include "display.h"
#include <stdio.h>
#include <iostream>

using namespace std;

Display::Display() {
    Display(1280, 800);
}

Display::Display(int width, int height) {
    this->height = height;
    this->width = width;

    if (this->SDLInit(&(this->window), &(this->renderer)) != 0) {
        cerr << "fails to open windows: " << endl;
        exit(127);
    }

    this->texture = SDL_CreateTexture(this->renderer, SDL_PIXELFORMAT_IYUV, SDL_TEXTUREACCESS_STREAMING, width, height);
    if (this->texture == nullptr) {
        fprintf(stderr, "CreateTextureFromSurface failed: %s\n", SDL_GetError());
        exit(1);
    }
    SDL_SetRenderTarget(this->renderer, this->texture);
}

void Display::SetWindowSize() {
    SDL_SetWindowSize(this->window, this->width, this->height);
}

void Display::ShowFrame() {
    SDL_SetWindowSize(this->window, this->width, this->height);
    SDL_RenderClear(this->renderer);
    // SDL_SetWindowTitle(this->window, "test");
    SDL_SetRenderDrawColor(renderer, 0, 0, 0, 255);
    SDL_RenderCopy(this->renderer, this->texture, nullptr, nullptr);
    SDL_RenderPresent(this->renderer);

    return;
}

bool Display::SetTexture(AVFrame *frame) {
    if (frame->linesize[0] > 0 && frame->linesize[1] > 0 && frame->linesize[2] > 0) {
        SDL_UpdateYUVTexture(this->texture, NULL, frame->data[0], frame->linesize[0],
                                                frame->data[1], frame->linesize[1],
                                                frame->data[2], frame->linesize[2]);
    } else if (frame->linesize[0] < 0 && frame->linesize[1] < 0 && frame->linesize[2] < 0) {
        SDL_UpdateYUVTexture(this->texture, NULL, frame->data[0] + frame->linesize[0] * (frame->height - 1), -frame->linesize[0],
                                                frame->data[1] + frame->linesize[1] * (AV_CEIL_RSHIFT(frame->height, 1) - 1), -frame->linesize[1],
                                                frame->data[2] + frame->linesize[2] * (AV_CEIL_RSHIFT(frame->height, 1) - 1), -frame->linesize[2]);
    } else {
        av_log(NULL, AV_LOG_ERROR, "Mixed negative and positive linesizes are not supported\n");
        return -1;
    }

    return true;
}

Display::~Display() {
    SDL_DestroyTexture(this->texture);
    SDL_DestroyWindow(this->window);
}

int Display::SDLInit(SDL_Window **window, SDL_Renderer **renderer) {
    *window = SDL_CreateWindow("悠悠是个小笨蛋", SDL_WINDOWPOS_UNDEFINED, SDL_WINDOWPOS_UNDEFINED, this->width, this->height, SDL_WINDOW_SHOWN|SDL_WINDOW_RESIZABLE);
    if (*window == nullptr) {
        SDL_Log("Unable to initialize window: %s", SDL_GetError());
        SDL_Quit();
        return 1;
    }
    SDL_SetHint(SDL_HINT_RENDER_SCALE_QUALITY, "linear");
    *renderer = SDL_CreateRenderer(*window, -1, SDL_RENDERER_ACCELERATED | SDL_RENDERER_PRESENTVSYNC);
    if (*renderer == nullptr) {
        SDL_DestroyWindow(*window);
        SDL_Log("Unable to initialize render: %s", SDL_GetError());
        SDL_Quit();
        return 1;
    }

    return 0;
}