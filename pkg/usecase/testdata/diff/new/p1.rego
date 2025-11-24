package ingest.color

alert contains {} if {
    not ignore
}

ignore if {
    input.name == "test"
    input.color == "red"
}
