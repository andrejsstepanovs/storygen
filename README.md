# StoryGen

Generates stories based on a given input text.


Download latest release binary file from [releases](https://github.com/andrejsstepanovs/storygen/tags).
```
# copy env file
cp app.env.example app.env
```

Define target audience and other params to generate a story you want.

```
# run the app from terminal
./storygen story create "a story about blue dog and cat with big ears"
./storygen story create "about Raichu who learned that not all Pokemonds know how to use electricity"
```


## Under the hood - Story Creation process

Each step builds on top of all previous steps. 
The process is as follows:

1. Use and store **user suggestion**
2. Retrieve random Story **Structure** from [structure.go](pkg/story/structure.go)
3. Calculate **Chapter Count** based on configured story length
4. Calculate **Chapter Word Count** (last chapter shorter)
5. Pick Story **Time Period** (based on user input) from [time_periods.go](pkg/story/time_periods.go)
6. Pick all good matching Story **Morales** 
7. Select **random X morales** from picked (count defined in configuration)
8. Figure out Story **Protagonists** and **their voices** following pattern described in [protagonists.go](pkg/story/protagonists.go)
9. Figure out Story **Villain**
10. Figure out **Villain Voice** 
11. Figure out Story **Location**
12. Draft a general Story **Plot Plan**
13. Describe and save Story **Summary**
14. Figure out Story **Chapter Titles**
15. For each chapter (with previous chapters content):
    - Create **Chapter Text**
17. Figure out Story **Title**
18. Saves story as json file
19. **Refining process** starts and loops as many times as configured:
20. Locate Story **Logical Problems** and pinpoint to specific chapter
21. **Sort problems** so first problem is for chapter 1 and last one is for last
22. For each problem: Suggest possible **Problem Fixes**
23. For each problem suggestion: **Adjust Story Chapter** using suggested fix
24. GOTO 19; if another loop needed
25. If configuration wants not English, then **Translate Story** by:
    - Translate Story Title
    - For each chapter:
      - Translate Chapter Title
      - Translate Chapter Text
    - Translate word "Chapter"
    - Translate word "The End"
    - Save translated story as new json file
26. **Text to speech** process. Input is finalized, ready to read story text.
    - Prepare **speech parameters** (mp3 filename, voice, speed, model, tone, affect, pacing, emotions, pauses)
    - Split text by chapters
    - For each chapter:
      - Split text into **chunks** (somewhat complex logic here).
      - **Convert** Chapter Chunk Text into **audio file**
    - **Combine audio files** into one
    - Remove all temporary files
27. If **audio post-processing** is enabled (recommended)
    - Remove **loud noises** from audio (because OpenAI often includes it)
    - Remove **long silences** from audio (another known OpenAI issue)
    - Delete original audio file
28. Present user with mp3 file of the story
