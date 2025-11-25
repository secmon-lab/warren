package ingest.color

alert contains {} if {
    not ignore
}

ignore if {
    input.name == "test"
}

ignore if {
    input.color == "blue"
}
