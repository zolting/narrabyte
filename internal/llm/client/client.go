package client

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"log"
	"narrabyte/internal/llm/tools"
	"strconv"
)

type OpenAIClient struct {
	ChatModel openai.ChatModel
	//GitToolsService tools.GitToolsService
	Key string
}

func NewOpenAIClient(ctx context.Context, key string) (*OpenAIClient, error) {
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey: key,
		Model:  "gpt-5-mini",
	})

	if err != nil {
		log.Printf("Error creating OpenAI client: %v", err)
		return nil, err
	}

	return &OpenAIClient{*model, key}, err
}

//La fonction essaie d'appeler un modèle de chat avec un outil d'addition, d'exécuter le flux "LLM -> Tools" puis de retourner le dernier message produit. Elle prépare les messages, attache l'outil au modèle, construit une chaîne composée d'un nœud LLM puis d'un nœud Tools, compile et invoque l'agent.
//
//Étapes détaillées :
//
//
//Infère un outil nommé add_tool via utils.InferTool.
//Construit la liste messages (System + User) avec la question d'addition.
//Appelle addTool.Info(ctx) mais ignore l'erreur (ligne actuelle).
//Lie les métadonnées de l'outil au modèle via o.ChatModel.BindForcedTools.
//Crée un toolsNode contenant l'outil obtenu.
//Crée une chain et y ajoute un nœud ChatModel (llm-plan) puis le toolsNode.
//Compile la chaîne en agent.
//Invoque agent.Invoke(ctx, messages) pour exécuter la composition.
//Vérifie que outMsg n'est pas nil, puis tente de renvoyer le contenu du dernier message avec outMsg[len(outMsg)-1].Content.
//Points problématiques / risques (à corriger) :
//
//
//info, _ := addTool.Info(ctx) ignore l'erreur : si info est nil, la liaison et l'exécution échoueront.
//outMsg est de type *[]schema.Message (pointeur vers slice) — on ne peut pas faire outMsg[len(outMsg)-1]. Il faut d'abord vérifier outMsg != nil, puis faire msgs := *outMsg et indexer msgs[len(msgs)-1].
//Il manque des contrôles d'erreur supplémentaires (vérifier la compilation, la réussite de la liaison des outils, et la longueur du slice avant d'indexer).

// Merci chat pour le resume :)

// Etant donne que c'est une demo simple, le context est Background() mais il faudrait peut-etre mettre un timeout
func (o *OpenAIClient) InvokeAdditionDemo(ctx context.Context, a, b int) (string, error) {
	addTool, err := utils.InferTool("add_tool", "adds two integers and gives the result", tools.Add)
	if err != nil {
		log.Printf("Error inferring tool: %v", err)
		return "", err
	}

	messages := []*schema.Message{
		schema.SystemMessage("You are a helpful assistant that can perform addition using the provided tool."),
		schema.UserMessage("What is the sum of" + strconv.Itoa(a) + "and" + strconv.Itoa(b) + "?"),
	}

	info, _ := addTool.Info(ctx)
	if err := o.ChatModel.BindForcedTools([]*schema.ToolInfo{info}); err != nil {
		log.Printf("Error binding tools: %v", err)
		return "", err
	}

	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{addTool},
	})
	if err != nil {
		log.Printf("Error creating tools node: %v", err)
		return "", err
	}

	chain := compose.NewChain[[]*schema.Message, []*schema.Message]()
	chain.AppendChatModel(&o.ChatModel, compose.WithNodeName("llm-plan"))
	chain.AppendToolsNode(toolsNode, compose.WithNodeName("tools"))

	agent, err := chain.Compile(ctx)
	if err != nil {
		log.Printf("Error compiling chain: %v", err)
		return "", err
	}

	outMsg, err := agent.Invoke(ctx, messages)
	if err != nil {
		log.Printf("Error invoking agent: %v", err)
		return "", err
	}
	if outMsg == nil {
		return "", fmt.Errorf("agent returned no message")
	}

	return outMsg[len(outMsg)-1].Content, nil
}
