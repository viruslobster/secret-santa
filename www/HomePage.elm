module HomePage exposing (main)

import Html exposing (..)
import Html.Attributes exposing (..)


view model =
    div [ class "jumbotron" ]
        [ h1 [] [ text "🎅 Welcome to Santa's Workshop! 🎄" ]
        , p []
            [ text "Santa's Workshop Inc. (stock symbol "
            , strong [] [ text "HOHO" ]
            , text <|
                """
                ) is a magical toy manufacturing and gift
                delivery service with an emphasis on spreading
                joy to children worldwide.
                """
            ]
        ]


main =
    view "dummy model"
