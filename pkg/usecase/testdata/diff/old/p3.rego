package ingest.color

alert contains {} if {
    not ignore
}

ignore if {
    input.color == "blue"
}
