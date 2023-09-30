set CGO_CXXFLAGS="--std=c++11"
set CGO_CPPFLAGS=-IC:\opencv\build\install\include
set CGO_LDFLAGS=-LC:\opencv\build\install\x64\mingw\lib -lopencv_core480 -lopencv_face480 -lopencv_videoio480 -lopencv_imgproc480 -lopencv_highgui480 -lopencv_imgcodecs480 -lopencv_objdetect480 -lopencv_features2d480 -lopencv_video480 -lopencv_dnn480 -lopencv_xfeatures2d480 -lopencv_plot480 -lopencv_tracking480 -lopencv_img_hash480
go build -o .\release\core-windows-amd64.exe