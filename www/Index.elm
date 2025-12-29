module Index exposing (main)

import Browser
import Html exposing (..)
import Html.Attributes exposing (..)


main : Program () Model Msg
main =
    Browser.document
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        }


type alias Model =
    {}


type Msg
    = Foo


init : () -> ( Model, Cmd Msg )
init flags =
    ( {}, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    ( model, Cmd.none )


view : Model -> Browser.Document Msg
view model =
    { title = "Secret Santa", body = viewBody model }


viewBody : Model -> List (Html Msg)
viewBody model =
    [ div
        [ class "jumbotron" ]
        [ h1 [] [ text "🎅 Secret Santa 🎄" ] ]
    , a [ href "/server" ] [ text "Go to server" ]
    ]
