port module ServerPage exposing (main)

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


port subscribeThread : ThreadId -> Cmd a


port recieveChatData : (( ThreadId, String ) -> a) -> Sub a


type alias Flags =
    { name : String, userId : Int, thread : ThreadId, server : Int }


main : Program Flags Model Msg
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
    , thread : ThreadId
    , server : Int
    , name : String
    , userId : Int
    , user : String
    , chats : List ChatMessage
    , chatDraft : String
    , error : String
    }


type Msg
    = LinkClicked Browser.UrlRequest
    | UrlChanged Url.Url
    | SendChat ChatMessage ThreadId
    | SendChatResponse (Result Http.Error ())
    | UpdateChatDraft String
    | RecieveChatThread (Result Http.Error ChatThread)
    | RecieveChatData ( ThreadId, String )


init : Flags -> Url.Url -> Nav.Key -> ( Model, Cmd Msg )
init flags url key =
    ( { key = key
      , url = url
      , name = flags.name
      , userId = flags.userId
      , user = "You"
      , thread = flags.thread
      , server = flags.server
      , chats = []
      , chatDraft = ""
      , error = ""
      }
    , Cmd.batch [ fetchThread flags.userId flags.thread, subscribeThread flags.thread ]
    )


httpError : Http.Error -> String
httpError error =
    case error of
        Http.BadStatus status ->
            "HTTP " ++ String.fromInt status ++ ". Try reloading or logging in again."

        _ ->
            "Please check your connection and reload."


fetchThread : Int -> ThreadId -> Cmd Msg
fetchThread userId id =
    Http.get
        { url = "/api/thread?id=" ++ String.fromInt id
        , expect = Http.expectJson RecieveChatThread (chatThreadDecoder userId)
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
                    ( model, Nav.load (Url.toString url) )

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
                        { model | chats = model.chats ++ [ chat ], chatDraft = "", error = "" }

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

                Err error ->
                    ( { model | error = "Message was not sent: " ++ httpError error }, Cmd.none )

        RecieveChatThread resp ->
            case resp of
                Ok thread ->
                    ( { model | chats = thread.chats }, Cmd.none )

                Err error ->
                    ( { model | error = "Could not load chat: " ++ httpError error }, Cmd.none )

        UpdateChatDraft draft ->
            ( { model | chatDraft = draft }, Cmd.none )

        RecieveChatData ( thread_id, data ) ->
            let
                _ =
                    thread_id

                newModel =
                    case Decode.decodeString (chatDecoder model.userId) data of
                        Ok chat ->
                            { model | chats = updateChats model.chats chat }

                        Err error ->
                            let
                                _ =
                                    Debug.log "Decode chat error: " error
                            in
                            model
            in
            ( newModel, Cmd.none )


{-| Incorporate a new chat message into a thread. This is a little involved because
a message might already be in the list, in which case its status just needs to be updated
-}
updateChats : List ChatMessage -> ChatMessage -> List ChatMessage
updateChats chats chat =
    case chat.status of
        Unsent ->
            chats ++ [ chat ]

        Error ->
            chats ++ [ chat ]

        Sent sent_at ->
            updateSentChat chats chat.message chat.user sent_at


{-| Incorporates a sent chat into `chats`. If the chat is from another user, it just gets appended.
If the chat is already in `chats` as a status of Unsent or Error, we update the status to Sent ts.
This function is idempotent.
-}
updateSentChat : List ChatMessage -> String -> String -> Time.Posix -> List ChatMessage
updateSentChat chats message user ts =
    case chats of
        [] ->
            [ { message = message, user = user, status = Sent ts } ]

        first :: rest ->
            let
                continue =
                    first :: updateSentChat rest message user ts

                updateStatus =
                    { first | status = Sent ts } :: rest
            in
            case first.status of
                Unsent ->
                    if first.message == message && first.user == user then
                        updateStatus

                    else
                        continue

                Sent sent_at ->
                    if first.message == message && first.user == user && ts == sent_at then
                        first :: rest

                    else
                        continue

                Error ->
                    if first.message == message && first.user == user then
                        updateStatus

                    else
                        continue


subscriptions : Model -> Sub Msg
subscriptions _ =
    recieveChatData RecieveChatData


view : Model -> Browser.Document Msg
view model =
    { title = "Secret Santa", body = viewBody model }


viewBody : Model -> List (Html Msg)
viewBody model =
    [ div [ class "jumbotron" ]
        [ h1 [] [ text "🎅 Secret Santa 🎄" ]
        , p [] [ text ("Hello " ++ model.name) ]
        , a [ href ("/server/" ++ String.fromInt model.server ++ "/santa") ] [ text "Your Secret Santa Chat" ]
        , text " - "
        , a [ href ("/server/" ++ String.fromInt model.server ++ "/giftee") ] [ text "Your Giftee Chat" ]
        ]
    , h2 []
        [ text
            (case String.split "/" model.url.path of
                [ "", "server", _, "santa" ] ->
                    "Your Secret Santa Chat"

                [ "", "server", _, "giftee" ] ->
                    "Your Giftee Chat"

                _ ->
                    "Chat not found"
            )
        ]
    , div [ attribute "role" "alert" ] [ text model.error ]
    , div [] [ viewChat model ]
    ]


viewChat : Model -> Html Msg
viewChat model =
    case String.split "/" model.url.path of
        [ "", "server", _, "santa" ] ->
            viewChatThread model

        [ "", "server", _, "giftee" ] ->
            viewChatThread model

        _ ->
            text "404"


viewChatThread : Model -> Html Msg
viewChatThread model =
    let
        thread =
            model.thread
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
    let
        status =
            case chat.status of
                Unsent ->
                    "unsent"

                Sent ts ->
                    "sent " ++ formatTime Time.utc ts

                Error ->
                    "error"
    in
    div [ class "chat-message" ]
        [ text (chat.user ++ ": " ++ chat.message ++ " (" ++ status ++ ")") ]


formatTime : Time.Zone -> Time.Posix -> String
formatTime zone time =
    String.fromInt (Time.toHour zone time)
        ++ ":"
        ++ String.fromInt (Time.toMinute zone time)


encodeChatRequest : ChatMessage -> ThreadId -> Encode.Value
encodeChatRequest chat thread =
    Encode.object
        [ ( "message", Encode.string chat.message )
        , ( "thread", Encode.int thread )
        ]


chatThreadDecoder : Int -> Decode.Decoder ChatThread
chatThreadDecoder userId =
    Decode.map2 ChatThread
        (Decode.field "id" Decode.int)
        (Decode.field "chats" (Decode.list (chatDecoder userId)))


chatDecoder : Int -> Decode.Decoder ChatMessage
chatDecoder userId =
    Decode.map3 ChatMessage
        (Decode.field "message" Decode.string)
        (Decode.field "user"
            (Decode.nullable
                (Decode.map2
                    (\id name ->
                        if id == userId then
                            "You"

                        else
                            name
                    )
                    (Decode.field "id" Decode.int)
                    (Decode.field "name" Decode.string)
                )
            )
            |> Decode.map (Maybe.withDefault "Santa")
        )
        (Decode.field "timestamp" Decode.int
            |> Decode.map (\s -> s * 1000)
            |> Decode.map Time.millisToPosix
            |> Decode.map Sent
        )
