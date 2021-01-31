# Prototype of swiping a region of a window to create its minimized thumbnail

rsnous on [Jan 28, 2021](https://twitter.com/rsnous/status/1354834356032860160): "maybe every time you want to switch away from a tab, the browser makes you sweep your mouse to select a meaningful index quote for that tab first."

And so vncfreethumb was born…

…except actually embedding a browser in a Go app is too hard, so this just displays all the images in the directory you provide as the first argument.

* cmd/server/ui.go implements the GUI
* cmd/server/main.go implements a VNC server to host the GUI
* rfb/rfb.go and rfb/image.go implement the relevant parts of the VNC (Remote Framebuffer) protocol

Press W, A, S, D to fold back parts of a window. Swipe a region with the right mouse button to fold back everything outside of it. Click the right mouse button to toggle all folds.
