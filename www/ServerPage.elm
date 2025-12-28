module ServerPage exposing (main)

import Browser
import Browser.Navigation as Nav
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events as Events
import Http
import Json.Decode as Decode
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


type Msg
    = LinkClicked Browser.UrlRequest
    | UrlChanged Url.Url
    | SendChat ChatMessage ThreadId
    | SendChatResponse (Result Http.Error ())
    | UpdateChatDraft String
    | RecieveChatThread (Result Http.Error ChatThread)


init : () -> Url.Url -> Nav.Key -> ( Model, Cmd Msg )
init flags url key =
    ( { key = key
      , url = url
      , user = "Mr. Foo"
      , chats = []
      , chatDraft = ""
      }
    , Http.get
        { url = "/api/thread?id=0"
        , expect = Http.expectJson RecieveChatThread chatThreadDecoder
        }
    )


type alias ChatMessage =
    { message : String
    , user : String
    , status : ChatStatus
    }


type ChatStatus
    = Unsent
    | Sent Time.Posix
    | Error


type alias ThreadId =
    Int


type alias ChatThread =
    { id : ThreadId
    , chats : List ChatMessage
    }


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

        SendChat chat thread ->
            if chat.message == "" then
                ( model, Cmd.none )

            else
                let
                    newModel =
                        { model | chats = model.chats ++ [ chat ], chatDraft = "" }

                    request =
                        Http.post
                            { url = "/api/chat/publish"
                            , body = Http.jsonBody (encodeChatRequest chat thread)
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

        RecieveChatThread resp ->
            case resp of
                Ok thread ->
                    ( { model | chats = thread.chats }, Cmd.none )

                Err error ->
                    let
                        _ =
                            Debug.log "Recieve chat thread: " error
                    in
                    ( model, Cmd.none )

        UpdateChatDraft draft ->
            ( { model | chatDraft = draft }, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none


view : Model -> Browser.Document Msg
view model =
    { title = "Secret Santa", body = viewBody model }


viewBody : Model -> List (Html Msg)
viewBody model =
    [ div [ class "jumbotron" ]
        [ h1 [] [ text "🎅 Secret Santa 🎄" ]
        , a [ href "/server" ] [ text "Group Chat" ]
        , text " - "
        , a [ href "/server/secret-santa" ] [ text "Your Secret Santa Chat" ]
        , text " - "
        , a [ href "/server/giftee" ] [ text "Your Giftee Chat" ]
        ]
    , div [] [ viewChat model ]
    ]


viewChat : Model -> Html Msg
viewChat model =
    case model.url.path of
        "/server" ->
            viewChatThread model

        "/server/secret-santa" ->
            text "keep it secret, keep it safe"

        "/server/giftee" ->
            text "gimmie"

        _ ->
            text "404"


viewChatThread : Model -> Html Msg
viewChatThread model =
    let
        thread =
            0
    in
    div [ class "chat-thread" ]
        [ div [] (List.map viewChatMessage model.chats)
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
                    thread
                )
            ]
            [ text "Send" ]
        ]


viewChatMessage : ChatMessage -> Html Msg
viewChatMessage chat =
    div [ class "chat-message" ] [ text (chat.user ++ ": " ++ chat.message) ]


encodeChatRequest : ChatMessage -> ThreadId -> Encode.Value
encodeChatRequest chat thread =
    Encode.object
        [ ( "message", Encode.string chat.message )
        , ( "user", Encode.string chat.user )
        , ( "thread", Encode.int thread )
        ]


encodeChat : ChatMessage -> Encode.Value
encodeChat chat =
    Encode.object
        [ ( "message", Encode.string chat.message )
        , ( "user", Encode.string chat.user )
        ]


chatThreadDecoder : Decode.Decoder ChatThread
chatThreadDecoder =
    Decode.map2 ChatThread
        (Decode.field "id" Decode.int)
        (Decode.field "chats" (Decode.list chatDecoder))


chatDecoder : Decode.Decoder ChatMessage
chatDecoder =
    Decode.map3 ChatMessage
        (Decode.field "message" Decode.string)
        (Decode.field "user" Decode.string)
        (Decode.field "timestamp" Decode.int
            |> Decode.map (\s -> s * 1000)
            |> Decode.map Time.millisToPosix
            |> Decode.map Sent
        )
