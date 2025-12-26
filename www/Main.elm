module Main exposing (main)

import Browser
import Browser.Navigation as Nav
import Html exposing (..)
import Html.Attributes exposing (..)
import Url


main : Program () Model Msg
main =
    Browser.application
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        , onUrlChange = UrlChanged
        , onUrlRequest = LinkClicked
        }


type alias Model =
    { key : Nav.Key, url : Url.Url }


init : () -> Url.Url -> Nav.Key -> ( Model, Cmd Msg )
init flags url key =
    ( Model key url, Cmd.none )


type Msg
    = LinkClicked Browser.UrlRequest
    | UrlChanged Url.Url


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        LinkClicked urlRequest ->
            case urlRequest of
                Browser.Internal url ->
                    ( model, Nav.pushUrl model.key (Url.toString url) )

                Browser.External href ->
                    ( model, Nav.load href )

        UrlChanged url ->
            ( { model | url = url }, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none


view : Model -> Browser.Document Msg
view model =
    { title = "Secret Santa", body = body model.url }


body : Url.Url -> List (Html Msg)
body url =
    [ div [ class "jumbotron" ]
        [ h1 [] [ text "🎅 Secret Santa 🎄" ]
        , a [ href "/group" ] [ text "Group Chat" ]
        , text " - "
        , a [ href "/secret-santa" ] [ text "Your Secret Santa Chat" ]
        , text " - "
        , a [ href "/giftee" ] [ text "Your Giftee Chat" ]
        ]
    , div []
        [ case url.path of
            "/group" ->
                text "we in the group"

            "/secret-santa" ->
                text "keep it secret, keep it safe"

            "/giftee" ->
                text "gimmie"

            _ ->
                text "404"
        ]
    ]
