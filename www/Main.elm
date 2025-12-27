module Main exposing (main)

import Browser
import Browser.Navigation as Nav
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events as Events
import Http
import Json.Encode as Encode
import Time
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
    { key : Nav.Key
    , url : Url.Url
    , user : String
    , chats : List ChatMessage
    , chatDraft : String
    }


type alias ChatMessage =
    { message : String
    , user : String
    , status : ChatStatus
    }


type ChatStatus
    = Unsent
    | Sent Time.Posix
    | Error


init : () -> Url.Url -> Nav.Key -> ( Model, Cmd Msg )
init flags url key =
    ( { key = key
      , url = url
      , user = "Mr. Foo"
      , chats = []
      , chatDraft = ""
      }
    , Cmd.none
    )


type Msg
    = LinkClicked Browser.UrlRequest
    | UrlChanged Url.Url
    | SendChat ChatMessage
    | SendChatResponse (Result Http.Error ())
    | UpdateChatDraft String


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

        SendChat chat ->
            let
                newModel =
                    { model | chats = model.chats ++ [ chat ], chatDraft = "" }

                request =
                    Http.post
                        { url = "/api/sendChat"
                        , body = Http.jsonBody (encodeChat chat)
                        , expect = Http.expectWhatever SendChatResponse
                        }
            in
            ( newModel, request )

        SendChatResponse resp ->
            case resp of
                -- TODO: update the unsent chat message to sent
                Ok () ->
                    ( model, Cmd.none )

                -- TODO: retry or add an error to the chat
                Err error ->
                    let
                        _ =
                            Debug.log "Chat send error" error
                    in
                    ( model, Cmd.none )

        UpdateChatDraft draft ->
            ( { model | chatDraft = draft }, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none


view : Model -> Browser.Document Msg
view model =
    { title = "Secret Santa", body = body model }


body : Model -> List (Html Msg)
body model =
    [ div [ class "jumbotron" ]
        [ h1 [] [ text "🎅 Secret Santa 🎄" ]
        , a [ href "/group" ] [ text "Group Chat" ]
        , text " - "
        , a [ href "/secret-santa" ] [ text "Your Secret Santa Chat" ]
        , text " - "
        , a [ href "/giftee" ] [ text "Your Giftee Chat" ]
        ]
    , div [] [ chatView model ]
    ]


chatView : Model -> Html Msg
chatView model =
    case model.url.path of
        "/group" ->
            chatThread model

        "/secret-santa" ->
            text "keep it secret, keep it safe"

        "/giftee" ->
            text "gimmie"

        "/" ->
            text ""

        _ ->
            text "404"


chatThread : Model -> Html Msg
chatThread model =
    div [ class "chat-thread" ]
        [ div [] (List.map chatMessage model.chats)
        , textarea
            [ value model.chatDraft, Events.onInput UpdateChatDraft ]
            []
        , button
            [ Events.onClick
                (SendChat
                    { message = model.chatDraft
                    , user = model.user
                    , status = Unsent
                    }
                )
            ]
            [ text "Send" ]
        ]


chatMessage : ChatMessage -> Html Msg
chatMessage chat =
    div [ class "chat-message" ] [ text (chat.user ++ ": " ++ chat.message) ]


encodeChat : ChatMessage -> Encode.Value
encodeChat chat =
    Encode.object
        [ ( "message", Encode.string chat.message )
        , ( "user", Encode.string chat.user )
        ]
