# fancyfmt
A formatter for Go code with bells and whistles

## Installation

* Use 

    ```shell script
    go get github.com/sirkon/fancyfmt
    ```

    to get a library as a dependency of your custom formatter.

* Type

    ```shell script
    go install github.com/sirkon/cmd/fancyfmt@latest
    ```
  
    to install a ready to use formatter

## Description

[fancyfmt](https://github.com/sirkon/fancyfmt) is a

* library to make gofmt compliant formatters
* a ready to use formatter

### Brief functionality
 

* Imports grouping and sorting. Import paths are:
    * Joined in one import declaration (except "C", which is always alone)
    * Splitted in groups (can be tweaked with custom imports grouper)
    * Sorted lexicographically within each group
* Provides default formatting for
    * Multiline functions declarations
    * Multiline calls
    * Multiline composite literals, slices and arrays get a special care at that
    * Multiline chaining
    
### Screencast

[![usage screencast](https://i9.ytimg.com/vi/WmqG-OTyF6g/mq2.jpg?sqp=CODsvvkF&rs=AOn4CLBgkcpGPMAak_SacamvPV9uXDA-eA)](https://youtu.be/WmqG-OTyF6g)

## Remarks

* fancyfmt stores a cache of packages from standard library in a `os.TempDir()` directory. This was done to speedup 
things as `package.Load(cfg, "std")` is slow, about 0.2s on my machine, cached access is about ten times faster. You
may notice a slugishness in case of the first formatting in the screencast, that is it. The further formats are much
faster.
* fancyfmt mutates `[]byte{…}` literals if they only have numbers replacing them with hex numbers.
* You may fix composite literals formatting (except the new line before the first item and after the last one) by
adding a comment after an element. 
