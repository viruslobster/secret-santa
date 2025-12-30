module Index exposing (main)

import Browser
import Browser.Navigation as Nav
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events as Events
import Http
import Json.Encode as Encode


main : Program () Model Msg
main =
    Browser.document
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        }


type alias Model =
    { username : String }


type Msg
    = Login String
    | LoginResponse (Result Http.Error ())
    | UserNameInput String


init : () -> ( Model, Cmd Msg )
init flags =
    ( { username = "" }, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        Login username ->
            ( model
            , Http.post
                { url = "/"
                , body = Http.jsonBody (encodeLoginRequest username)
                , expect = Http.expectWhatever LoginResponse
                }
            )

        LoginResponse result ->
            case result of
                Ok () ->
                    ( model, Nav.load "/server" )

                Err error ->
                    let
                        _ =
                            Debug.log "Login error: " error
                    in
                    ( model, Cmd.none )

        UserNameInput username ->
            ( { model | username = username }, Cmd.none )


view : Model -> Browser.Document Msg
view model =
    { title = "Secret Santa", body = viewBody model }


viewBody : Model -> List (Html Msg)
viewBody model =
    [ div
        [ class "jumbotron" ]
        [ h1 [] [ text "🎅 Secret Santa 🎄" ] ]
    , text "Who are you?"
    , div
        []
        [ input [ Events.onInput UserNameInput ] []
        , button
            [ Events.onClick (Login model.username) ]
            [ text "login" ]
        ]
    ]


encodeLoginRequest : String -> Encode.Value
encodeLoginRequest username =
    Encode.object
        [ ( "username", Encode.string username )
        ]
